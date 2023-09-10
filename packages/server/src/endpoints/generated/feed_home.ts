export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 8620,
          "end": 8624
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8625,
              "end": 8630
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8633,
                "end": 8638
              }
            },
            "loc": {
              "start": 8632,
              "end": 8638
            }
          },
          "loc": {
            "start": 8625,
            "end": 8638
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notes",
              "loc": {
                "start": 8646,
                "end": 8651
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_list",
                    "loc": {
                      "start": 8665,
                      "end": 8674
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8662,
                    "end": 8674
                  }
                }
              ],
              "loc": {
                "start": 8652,
                "end": 8680
              }
            },
            "loc": {
              "start": 8646,
              "end": 8680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 8685,
                "end": 8694
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Reminder_full",
                    "loc": {
                      "start": 8708,
                      "end": 8721
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8705,
                    "end": 8721
                  }
                }
              ],
              "loc": {
                "start": 8695,
                "end": 8727
              }
            },
            "loc": {
              "start": 8685,
              "end": 8727
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 8732,
                "end": 8741
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Resource_list",
                    "loc": {
                      "start": 8755,
                      "end": 8768
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8752,
                    "end": 8768
                  }
                }
              ],
              "loc": {
                "start": 8742,
                "end": 8774
              }
            },
            "loc": {
              "start": 8732,
              "end": 8774
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 8779,
                "end": 8788
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Schedule_list",
                    "loc": {
                      "start": 8802,
                      "end": 8815
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8799,
                    "end": 8815
                  }
                }
              ],
              "loc": {
                "start": 8789,
                "end": 8821
              }
            },
            "loc": {
              "start": 8779,
              "end": 8821
            }
          }
        ],
        "loc": {
          "start": 8640,
          "end": 8825
        }
      },
      "loc": {
        "start": 8620,
        "end": 8825
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 32,
          "end": 34
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 32,
        "end": 34
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 35,
          "end": 45
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 35,
        "end": 45
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 46,
          "end": 56
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 46,
        "end": 56
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 57,
          "end": 62
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 57,
        "end": 62
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 63,
          "end": 68
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 63,
        "end": 68
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 69,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 88,
                  "end": 100
                }
              },
              "loc": {
                "start": 88,
                "end": 100
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 114,
                      "end": 130
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 111,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 101,
                "end": 136
              }
            },
            "loc": {
              "start": 81,
              "end": 136
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 148,
                  "end": 152
                }
              },
              "loc": {
                "start": 148,
                "end": 152
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 166,
                      "end": 174
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 174
                  }
                }
              ],
              "loc": {
                "start": 153,
                "end": 180
              }
            },
            "loc": {
              "start": 141,
              "end": 180
            }
          }
        ],
        "loc": {
          "start": 75,
          "end": 182
        }
      },
      "loc": {
        "start": 69,
        "end": 182
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 183,
          "end": 186
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 193,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 207,
                "end": 216
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 216
            }
          }
        ],
        "loc": {
          "start": 187,
          "end": 218
        }
      },
      "loc": {
        "start": 183,
        "end": 218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 250,
          "end": 258
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 265,
                "end": 277
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 288,
                      "end": 290
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 288,
                    "end": 290
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 299,
                      "end": 307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 299,
                    "end": 307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 316,
                      "end": 327
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 316,
                    "end": 327
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 336,
                      "end": 340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 336,
                    "end": 340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "pages",
                    "loc": {
                      "start": 349,
                      "end": 354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 369,
                            "end": 371
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 369,
                          "end": 371
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pageIndex",
                          "loc": {
                            "start": 384,
                            "end": 393
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 384,
                          "end": 393
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 406,
                            "end": 410
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 406,
                          "end": 410
                        }
                      }
                    ],
                    "loc": {
                      "start": 355,
                      "end": 420
                    }
                  },
                  "loc": {
                    "start": 349,
                    "end": 420
                  }
                }
              ],
              "loc": {
                "start": 278,
                "end": 426
              }
            },
            "loc": {
              "start": 265,
              "end": 426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 431,
                "end": 433
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 431,
              "end": 433
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 438,
                "end": 448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 438,
              "end": 448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 453,
                "end": 463
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 453,
              "end": 463
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 468,
                "end": 476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 468,
              "end": 476
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 481,
                "end": 490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 481,
              "end": 490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 495,
                "end": 507
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 495,
              "end": 507
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 512,
                "end": 524
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 512,
              "end": 524
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 529,
                "end": 541
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 529,
              "end": 541
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 546,
                "end": 549
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canComment",
                    "loc": {
                      "start": 560,
                      "end": 570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 560,
                    "end": 570
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 579,
                      "end": 586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 579,
                    "end": 586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 595,
                      "end": 604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 595,
                    "end": 604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 613,
                      "end": 622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 613,
                    "end": 622
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 631,
                      "end": 640
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 631,
                    "end": 640
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 649,
                      "end": 655
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 649,
                    "end": 655
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 664,
                      "end": 671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 664,
                    "end": 671
                  }
                }
              ],
              "loc": {
                "start": 550,
                "end": 677
              }
            },
            "loc": {
              "start": 546,
              "end": 677
            }
          }
        ],
        "loc": {
          "start": 259,
          "end": 679
        }
      },
      "loc": {
        "start": 250,
        "end": 679
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 680,
          "end": 682
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 680,
        "end": 682
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 683,
          "end": 693
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 683,
        "end": 693
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 694,
          "end": 704
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 694,
        "end": 704
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 705,
          "end": 714
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 705,
        "end": 714
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 715,
          "end": 726
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 715,
        "end": 726
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 727,
          "end": 733
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 743,
                "end": 753
              }
            },
            "directives": [],
            "loc": {
              "start": 740,
              "end": 753
            }
          }
        ],
        "loc": {
          "start": 734,
          "end": 755
        }
      },
      "loc": {
        "start": 727,
        "end": 755
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 756,
          "end": 761
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 775,
                  "end": 787
                }
              },
              "loc": {
                "start": 775,
                "end": 787
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 801,
                      "end": 817
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 798,
                    "end": 817
                  }
                }
              ],
              "loc": {
                "start": 788,
                "end": 823
              }
            },
            "loc": {
              "start": 768,
              "end": 823
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 835,
                  "end": 839
                }
              },
              "loc": {
                "start": 835,
                "end": 839
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 853,
                      "end": 861
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 850,
                    "end": 861
                  }
                }
              ],
              "loc": {
                "start": 840,
                "end": 867
              }
            },
            "loc": {
              "start": 828,
              "end": 867
            }
          }
        ],
        "loc": {
          "start": 762,
          "end": 869
        }
      },
      "loc": {
        "start": 756,
        "end": 869
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 870,
          "end": 881
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 870,
        "end": 881
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 882,
          "end": 896
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 882,
        "end": 896
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 897,
          "end": 902
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 897,
        "end": 902
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 903,
          "end": 912
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 903,
        "end": 912
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 913,
          "end": 917
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Tag_list",
              "loc": {
                "start": 927,
                "end": 935
              }
            },
            "directives": [],
            "loc": {
              "start": 924,
              "end": 935
            }
          }
        ],
        "loc": {
          "start": 918,
          "end": 937
        }
      },
      "loc": {
        "start": 913,
        "end": 937
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 938,
          "end": 952
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 938,
        "end": 952
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 953,
          "end": 958
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 953,
        "end": 958
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 959,
          "end": 962
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 969,
                "end": 978
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 969,
              "end": 978
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 983,
                "end": 994
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 983,
              "end": 994
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 999,
                "end": 1010
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 999,
              "end": 1010
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1015,
                "end": 1024
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1015,
              "end": 1024
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1029,
                "end": 1036
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1029,
              "end": 1036
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1041,
                "end": 1049
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1041,
              "end": 1049
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1054,
                "end": 1066
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1054,
              "end": 1066
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1071,
                "end": 1079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1071,
              "end": 1079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1084,
                "end": 1092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1084,
              "end": 1092
            }
          }
        ],
        "loc": {
          "start": 963,
          "end": 1094
        }
      },
      "loc": {
        "start": 959,
        "end": 1094
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1141,
          "end": 1143
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1141,
        "end": 1143
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 1144,
          "end": 1155
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1144,
        "end": 1155
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 1156,
          "end": 1162
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1156,
        "end": 1162
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 1163,
          "end": 1175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1163,
        "end": 1175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1176,
          "end": 1179
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canAddMembers",
              "loc": {
                "start": 1186,
                "end": 1199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1186,
              "end": 1199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 1204,
                "end": 1213
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1204,
              "end": 1213
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1218,
                "end": 1229
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1218,
              "end": 1229
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 1234,
                "end": 1243
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1234,
              "end": 1243
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1248,
                "end": 1257
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1248,
              "end": 1257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1262,
                "end": 1269
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1262,
              "end": 1269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1274,
                "end": 1286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1274,
              "end": 1286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1291,
                "end": 1299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1291,
              "end": 1299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 1304,
                "end": 1318
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1329,
                      "end": 1331
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1329,
                    "end": 1331
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1340,
                      "end": 1350
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1340,
                    "end": 1350
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1359,
                      "end": 1369
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1359,
                    "end": 1369
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 1378,
                      "end": 1385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1378,
                    "end": 1385
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 1394,
                      "end": 1405
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1394,
                    "end": 1405
                  }
                }
              ],
              "loc": {
                "start": 1319,
                "end": 1411
              }
            },
            "loc": {
              "start": 1304,
              "end": 1411
            }
          }
        ],
        "loc": {
          "start": 1180,
          "end": 1413
        }
      },
      "loc": {
        "start": 1176,
        "end": 1413
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1453,
          "end": 1455
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1453,
        "end": 1455
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1456,
          "end": 1466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1456,
        "end": 1466
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1467,
          "end": 1477
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1467,
        "end": 1477
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 1478,
          "end": 1482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1478,
        "end": 1482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "description",
        "loc": {
          "start": 1483,
          "end": 1494
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1483,
        "end": 1494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "dueDate",
        "loc": {
          "start": 1495,
          "end": 1502
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1495,
        "end": 1502
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 1503,
          "end": 1508
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1503,
        "end": 1508
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isComplete",
        "loc": {
          "start": 1509,
          "end": 1519
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1509,
        "end": 1519
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderItems",
        "loc": {
          "start": 1520,
          "end": 1533
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1540,
                "end": 1542
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1540,
              "end": 1542
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1547,
                "end": 1557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1547,
              "end": 1557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1562,
                "end": 1572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1562,
              "end": 1572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1577,
                "end": 1581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1577,
              "end": 1581
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1586,
                "end": 1597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1586,
              "end": 1597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1602,
                "end": 1609
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1602,
              "end": 1609
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1614,
                "end": 1619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1614,
              "end": 1619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1624,
                "end": 1634
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1624,
              "end": 1634
            }
          }
        ],
        "loc": {
          "start": 1534,
          "end": 1636
        }
      },
      "loc": {
        "start": 1520,
        "end": 1636
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reminderList",
        "loc": {
          "start": 1637,
          "end": 1649
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1656,
                "end": 1658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1656,
              "end": 1658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1663,
                "end": 1673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1663,
              "end": 1673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1678,
                "end": 1688
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1678,
              "end": 1688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusMode",
              "loc": {
                "start": 1693,
                "end": 1702
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 1713,
                      "end": 1719
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1734,
                            "end": 1736
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1734,
                          "end": 1736
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 1749,
                            "end": 1754
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1749,
                          "end": 1754
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 1767,
                            "end": 1772
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1767,
                          "end": 1772
                        }
                      }
                    ],
                    "loc": {
                      "start": 1720,
                      "end": 1782
                    }
                  },
                  "loc": {
                    "start": 1713,
                    "end": 1782
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 1791,
                      "end": 1803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1818,
                            "end": 1820
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1818,
                          "end": 1820
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1833,
                            "end": 1843
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1833,
                          "end": 1843
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 1856,
                            "end": 1868
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1887,
                                  "end": 1889
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1887,
                                "end": 1889
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 1906,
                                  "end": 1914
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1906,
                                "end": 1914
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 1931,
                                  "end": 1942
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1931,
                                "end": 1942
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 1959,
                                  "end": 1963
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1959,
                                "end": 1963
                              }
                            }
                          ],
                          "loc": {
                            "start": 1869,
                            "end": 1977
                          }
                        },
                        "loc": {
                          "start": 1856,
                          "end": 1977
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 1990,
                            "end": 1999
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2018,
                                  "end": 2020
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2018,
                                "end": 2020
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 2037,
                                  "end": 2042
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2037,
                                "end": 2042
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 2059,
                                  "end": 2063
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2059,
                                "end": 2063
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 2080,
                                  "end": 2087
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2080,
                                "end": 2087
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 2104,
                                  "end": 2116
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2139,
                                        "end": 2141
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2139,
                                      "end": 2141
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 2162,
                                        "end": 2170
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2162,
                                      "end": 2170
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2191,
                                        "end": 2202
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2191,
                                      "end": 2202
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2223,
                                        "end": 2227
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2223,
                                      "end": 2227
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2117,
                                  "end": 2245
                                }
                              },
                              "loc": {
                                "start": 2104,
                                "end": 2245
                              }
                            }
                          ],
                          "loc": {
                            "start": 2000,
                            "end": 2259
                          }
                        },
                        "loc": {
                          "start": 1990,
                          "end": 2259
                        }
                      }
                    ],
                    "loc": {
                      "start": 1804,
                      "end": 2269
                    }
                  },
                  "loc": {
                    "start": 1791,
                    "end": 2269
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 2278,
                      "end": 2286
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Schedule_common",
                          "loc": {
                            "start": 2304,
                            "end": 2319
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2301,
                          "end": 2319
                        }
                      }
                    ],
                    "loc": {
                      "start": 2287,
                      "end": 2329
                    }
                  },
                  "loc": {
                    "start": 2278,
                    "end": 2329
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2338,
                      "end": 2340
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2338,
                    "end": 2340
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2349,
                      "end": 2353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2349,
                    "end": 2353
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2362,
                      "end": 2373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2362,
                    "end": 2373
                  }
                }
              ],
              "loc": {
                "start": 1703,
                "end": 2379
              }
            },
            "loc": {
              "start": 1693,
              "end": 2379
            }
          }
        ],
        "loc": {
          "start": 1650,
          "end": 2381
        }
      },
      "loc": {
        "start": 1637,
        "end": 2381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2421,
          "end": 2423
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2421,
        "end": 2423
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "index",
        "loc": {
          "start": 2424,
          "end": 2429
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2424,
        "end": 2429
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "link",
        "loc": {
          "start": 2430,
          "end": 2434
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2430,
        "end": 2434
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "usedFor",
        "loc": {
          "start": 2435,
          "end": 2442
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2435,
        "end": 2442
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2443,
          "end": 2455
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2462,
                "end": 2464
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2462,
              "end": 2464
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2469,
                "end": 2477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2469,
              "end": 2477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 2482,
                "end": 2493
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2482,
              "end": 2493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2498,
                "end": 2502
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2498,
              "end": 2502
            }
          }
        ],
        "loc": {
          "start": 2456,
          "end": 2504
        }
      },
      "loc": {
        "start": 2443,
        "end": 2504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2546,
          "end": 2548
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2546,
        "end": 2548
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2549,
          "end": 2559
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2549,
        "end": 2559
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2560,
          "end": 2570
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2560,
        "end": 2570
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 2571,
          "end": 2580
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2571,
        "end": 2580
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 2581,
          "end": 2588
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2581,
        "end": 2588
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 2589,
          "end": 2597
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2589,
        "end": 2597
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 2598,
          "end": 2608
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2615,
                "end": 2617
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2615,
              "end": 2617
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 2622,
                "end": 2639
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2622,
              "end": 2639
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 2644,
                "end": 2656
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2644,
              "end": 2656
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 2661,
                "end": 2671
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2661,
              "end": 2671
            }
          }
        ],
        "loc": {
          "start": 2609,
          "end": 2673
        }
      },
      "loc": {
        "start": 2598,
        "end": 2673
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 2674,
          "end": 2685
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2692,
                "end": 2694
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2692,
              "end": 2694
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 2699,
                "end": 2713
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2699,
              "end": 2713
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 2718,
                "end": 2726
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2718,
              "end": 2726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 2731,
                "end": 2740
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2731,
              "end": 2740
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 2745,
                "end": 2755
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2745,
              "end": 2755
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 2760,
                "end": 2765
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2760,
              "end": 2765
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 2770,
                "end": 2777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2770,
              "end": 2777
            }
          }
        ],
        "loc": {
          "start": 2686,
          "end": 2779
        }
      },
      "loc": {
        "start": 2674,
        "end": 2779
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 2819,
          "end": 2825
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "FragmentSpread",
            "name": {
              "kind": "Name",
              "value": "Label_list",
              "loc": {
                "start": 2835,
                "end": 2845
              }
            },
            "directives": [],
            "loc": {
              "start": 2832,
              "end": 2845
            }
          }
        ],
        "loc": {
          "start": 2826,
          "end": 2847
        }
      },
      "loc": {
        "start": 2819,
        "end": 2847
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModes",
        "loc": {
          "start": 2848,
          "end": 2858
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2865,
                "end": 2871
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2882,
                      "end": 2884
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2882,
                    "end": 2884
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "color",
                    "loc": {
                      "start": 2893,
                      "end": 2898
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2893,
                    "end": 2898
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2907,
                      "end": 2912
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2907,
                    "end": 2912
                  }
                }
              ],
              "loc": {
                "start": 2872,
                "end": 2918
              }
            },
            "loc": {
              "start": 2865,
              "end": 2918
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 2923,
                "end": 2935
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2946,
                      "end": 2948
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2946,
                    "end": 2948
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2957,
                      "end": 2967
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2957,
                    "end": 2967
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2976,
                      "end": 2986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2976,
                    "end": 2986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminders",
                    "loc": {
                      "start": 2995,
                      "end": 3004
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3019,
                            "end": 3021
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3019,
                          "end": 3021
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3034,
                            "end": 3044
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3034,
                          "end": 3044
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3057,
                            "end": 3067
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3057,
                          "end": 3067
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3080,
                            "end": 3084
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3080,
                          "end": 3084
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3097,
                            "end": 3108
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3097,
                          "end": 3108
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "dueDate",
                          "loc": {
                            "start": 3121,
                            "end": 3128
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3121,
                          "end": 3128
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 3141,
                            "end": 3146
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3141,
                          "end": 3146
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 3159,
                            "end": 3169
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3159,
                          "end": 3169
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderItems",
                          "loc": {
                            "start": 3182,
                            "end": 3195
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3214,
                                  "end": 3216
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3214,
                                "end": 3216
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3233,
                                  "end": 3243
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3233,
                                "end": 3243
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3260,
                                  "end": 3270
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3260,
                                "end": 3270
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3287,
                                  "end": 3291
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3287,
                                "end": 3291
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3308,
                                  "end": 3319
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3308,
                                "end": 3319
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 3336,
                                  "end": 3343
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3336,
                                "end": 3343
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 3360,
                                  "end": 3365
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3360,
                                "end": 3365
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 3382,
                                  "end": 3392
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3382,
                                "end": 3392
                              }
                            }
                          ],
                          "loc": {
                            "start": 3196,
                            "end": 3406
                          }
                        },
                        "loc": {
                          "start": 3182,
                          "end": 3406
                        }
                      }
                    ],
                    "loc": {
                      "start": 3005,
                      "end": 3416
                    }
                  },
                  "loc": {
                    "start": 2995,
                    "end": 3416
                  }
                }
              ],
              "loc": {
                "start": 2936,
                "end": 3422
              }
            },
            "loc": {
              "start": 2923,
              "end": 3422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resourceList",
              "loc": {
                "start": 3427,
                "end": 3439
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3450,
                      "end": 3452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3450,
                    "end": 3452
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3461,
                      "end": 3471
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3461,
                    "end": 3471
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3480,
                      "end": 3492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3507,
                            "end": 3509
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3507,
                          "end": 3509
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3522,
                            "end": 3530
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3522,
                          "end": 3530
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3543,
                            "end": 3554
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3543,
                          "end": 3554
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3567,
                            "end": 3571
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3567,
                          "end": 3571
                        }
                      }
                    ],
                    "loc": {
                      "start": 3493,
                      "end": 3581
                    }
                  },
                  "loc": {
                    "start": 3480,
                    "end": 3581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resources",
                    "loc": {
                      "start": 3590,
                      "end": 3599
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3614,
                            "end": 3616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3614,
                          "end": 3616
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "index",
                          "loc": {
                            "start": 3629,
                            "end": 3634
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3629,
                          "end": 3634
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 3647,
                            "end": 3651
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3647,
                          "end": 3651
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "usedFor",
                          "loc": {
                            "start": 3664,
                            "end": 3671
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3664,
                          "end": 3671
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 3684,
                            "end": 3696
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3715,
                                  "end": 3717
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3715,
                                "end": 3717
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 3734,
                                  "end": 3742
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3734,
                                "end": 3742
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3759,
                                  "end": 3770
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3759,
                                "end": 3770
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3787,
                                  "end": 3791
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3787,
                                "end": 3791
                              }
                            }
                          ],
                          "loc": {
                            "start": 3697,
                            "end": 3805
                          }
                        },
                        "loc": {
                          "start": 3684,
                          "end": 3805
                        }
                      }
                    ],
                    "loc": {
                      "start": 3600,
                      "end": 3815
                    }
                  },
                  "loc": {
                    "start": 3590,
                    "end": 3815
                  }
                }
              ],
              "loc": {
                "start": 3440,
                "end": 3821
              }
            },
            "loc": {
              "start": 3427,
              "end": 3821
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3826,
                "end": 3828
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3826,
              "end": 3828
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3833,
                "end": 3837
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3833,
              "end": 3837
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3842,
                "end": 3853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3842,
              "end": 3853
            }
          }
        ],
        "loc": {
          "start": 2859,
          "end": 3855
        }
      },
      "loc": {
        "start": 2848,
        "end": 3855
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetings",
        "loc": {
          "start": 3856,
          "end": 3864
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3871,
                "end": 3877
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 3891,
                      "end": 3901
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3888,
                    "end": 3901
                  }
                }
              ],
              "loc": {
                "start": 3878,
                "end": 3907
              }
            },
            "loc": {
              "start": 3871,
              "end": 3907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 3912,
                "end": 3924
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3935,
                      "end": 3937
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3935,
                    "end": 3937
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3946,
                      "end": 3954
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3946,
                    "end": 3954
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3963,
                      "end": 3974
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3963,
                    "end": 3974
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "link",
                    "loc": {
                      "start": 3983,
                      "end": 3987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3983,
                    "end": 3987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3996,
                      "end": 4000
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3996,
                    "end": 4000
                  }
                }
              ],
              "loc": {
                "start": 3925,
                "end": 4006
              }
            },
            "loc": {
              "start": 3912,
              "end": 4006
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4011,
                "end": 4013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4011,
              "end": 4013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 4018,
                "end": 4040
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4018,
              "end": 4040
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnOrganizationProfile",
              "loc": {
                "start": 4045,
                "end": 4070
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4045,
              "end": 4070
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 4075,
                "end": 4087
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4098,
                      "end": 4100
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4098,
                    "end": 4100
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 4109,
                      "end": 4120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4109,
                    "end": 4120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 4129,
                      "end": 4135
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4129,
                    "end": 4135
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 4144,
                      "end": 4156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4144,
                    "end": 4156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 4165,
                      "end": 4168
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canAddMembers",
                          "loc": {
                            "start": 4183,
                            "end": 4196
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4183,
                          "end": 4196
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 4209,
                            "end": 4218
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4209,
                          "end": 4218
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 4231,
                            "end": 4242
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4231,
                          "end": 4242
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 4255,
                            "end": 4264
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4255,
                          "end": 4264
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 4277,
                            "end": 4286
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4277,
                          "end": 4286
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 4299,
                            "end": 4306
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4299,
                          "end": 4306
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 4319,
                            "end": 4331
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4319,
                          "end": 4331
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 4344,
                            "end": 4352
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4344,
                          "end": 4352
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 4365,
                            "end": 4379
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4398,
                                  "end": 4400
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4398,
                                "end": 4400
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4417,
                                  "end": 4427
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4417,
                                "end": 4427
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4444,
                                  "end": 4454
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4444,
                                "end": 4454
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 4471,
                                  "end": 4478
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4471,
                                "end": 4478
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4495,
                                  "end": 4506
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4495,
                                "end": 4506
                              }
                            }
                          ],
                          "loc": {
                            "start": 4380,
                            "end": 4520
                          }
                        },
                        "loc": {
                          "start": 4365,
                          "end": 4520
                        }
                      }
                    ],
                    "loc": {
                      "start": 4169,
                      "end": 4530
                    }
                  },
                  "loc": {
                    "start": 4165,
                    "end": 4530
                  }
                }
              ],
              "loc": {
                "start": 4088,
                "end": 4536
              }
            },
            "loc": {
              "start": 4075,
              "end": 4536
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 4541,
                "end": 4558
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "members",
                    "loc": {
                      "start": 4569,
                      "end": 4576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4591,
                            "end": 4593
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4591,
                          "end": 4593
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4606,
                            "end": 4616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4606,
                          "end": 4616
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4629,
                            "end": 4639
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4629,
                          "end": 4639
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 4652,
                            "end": 4659
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4652,
                          "end": 4659
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 4672,
                            "end": 4683
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4672,
                          "end": 4683
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 4696,
                            "end": 4701
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4720,
                                  "end": 4722
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4720,
                                "end": 4722
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4739,
                                  "end": 4749
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4739,
                                "end": 4749
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4766,
                                  "end": 4776
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4766,
                                "end": 4776
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4793,
                                  "end": 4797
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4793,
                                "end": 4797
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4814,
                                  "end": 4825
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4814,
                                "end": 4825
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 4842,
                                  "end": 4854
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4842,
                                "end": 4854
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 4871,
                                  "end": 4883
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4906,
                                        "end": 4908
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4906,
                                      "end": 4908
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 4929,
                                        "end": 4940
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4929,
                                      "end": 4940
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 4961,
                                        "end": 4967
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4961,
                                      "end": 4967
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 4988,
                                        "end": 5000
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4988,
                                      "end": 5000
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5021,
                                        "end": 5024
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canAddMembers",
                                            "loc": {
                                              "start": 5051,
                                              "end": 5064
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5051,
                                            "end": 5064
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 5089,
                                              "end": 5098
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5089,
                                            "end": 5098
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 5123,
                                              "end": 5134
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5123,
                                            "end": 5134
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 5159,
                                              "end": 5168
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5159,
                                            "end": 5168
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 5193,
                                              "end": 5202
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5193,
                                            "end": 5202
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 5227,
                                              "end": 5234
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5227,
                                            "end": 5234
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5259,
                                              "end": 5271
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5259,
                                            "end": 5271
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 5296,
                                              "end": 5304
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5296,
                                            "end": 5304
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 5329,
                                              "end": 5343
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 5374,
                                                    "end": 5376
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5374,
                                                  "end": 5376
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 5405,
                                                    "end": 5415
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5405,
                                                  "end": 5415
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 5444,
                                                    "end": 5454
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5444,
                                                  "end": 5454
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 5483,
                                                    "end": 5490
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5483,
                                                  "end": 5490
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 5519,
                                                    "end": 5530
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5519,
                                                  "end": 5530
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5344,
                                              "end": 5556
                                            }
                                          },
                                          "loc": {
                                            "start": 5329,
                                            "end": 5556
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5025,
                                        "end": 5578
                                      }
                                    },
                                    "loc": {
                                      "start": 5021,
                                      "end": 5578
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4884,
                                  "end": 5596
                                }
                              },
                              "loc": {
                                "start": 4871,
                                "end": 5596
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5613,
                                  "end": 5625
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 5648,
                                        "end": 5650
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5648,
                                      "end": 5650
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5671,
                                        "end": 5679
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5671,
                                      "end": 5679
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5700,
                                        "end": 5711
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5700,
                                      "end": 5711
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5626,
                                  "end": 5729
                                }
                              },
                              "loc": {
                                "start": 5613,
                                "end": 5729
                              }
                            }
                          ],
                          "loc": {
                            "start": 4702,
                            "end": 5743
                          }
                        },
                        "loc": {
                          "start": 4696,
                          "end": 5743
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5756,
                            "end": 5759
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 5778,
                                  "end": 5787
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5778,
                                "end": 5787
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 5804,
                                  "end": 5813
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5804,
                                "end": 5813
                              }
                            }
                          ],
                          "loc": {
                            "start": 5760,
                            "end": 5827
                          }
                        },
                        "loc": {
                          "start": 5756,
                          "end": 5827
                        }
                      }
                    ],
                    "loc": {
                      "start": 4577,
                      "end": 5837
                    }
                  },
                  "loc": {
                    "start": 4569,
                    "end": 5837
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5846,
                      "end": 5848
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5846,
                    "end": 5848
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5857,
                      "end": 5867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5857,
                    "end": 5867
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5876,
                      "end": 5886
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5876,
                    "end": 5886
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5895,
                      "end": 5899
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5895,
                    "end": 5899
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 5908,
                      "end": 5919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5908,
                    "end": 5919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 5928,
                      "end": 5940
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5928,
                    "end": 5940
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5949,
                      "end": 5961
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5976,
                            "end": 5978
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5976,
                          "end": 5978
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 5991,
                            "end": 6002
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5991,
                          "end": 6002
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 6015,
                            "end": 6021
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6015,
                          "end": 6021
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 6034,
                            "end": 6046
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6034,
                          "end": 6046
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 6059,
                            "end": 6062
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canAddMembers",
                                "loc": {
                                  "start": 6081,
                                  "end": 6094
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6081,
                                "end": 6094
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 6111,
                                  "end": 6120
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6111,
                                "end": 6120
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 6137,
                                  "end": 6148
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6137,
                                "end": 6148
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 6165,
                                  "end": 6174
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6165,
                                "end": 6174
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 6191,
                                  "end": 6200
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6191,
                                "end": 6200
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 6217,
                                  "end": 6224
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6217,
                                "end": 6224
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 6241,
                                  "end": 6253
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6241,
                                "end": 6253
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 6270,
                                  "end": 6278
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6270,
                                "end": 6278
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 6295,
                                  "end": 6309
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 6332,
                                        "end": 6334
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6332,
                                      "end": 6334
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 6355,
                                        "end": 6365
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6355,
                                      "end": 6365
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 6386,
                                        "end": 6396
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6386,
                                      "end": 6396
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 6417,
                                        "end": 6424
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6417,
                                      "end": 6424
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 6445,
                                        "end": 6456
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6445,
                                      "end": 6456
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6310,
                                  "end": 6474
                                }
                              },
                              "loc": {
                                "start": 6295,
                                "end": 6474
                              }
                            }
                          ],
                          "loc": {
                            "start": 6063,
                            "end": 6488
                          }
                        },
                        "loc": {
                          "start": 6059,
                          "end": 6488
                        }
                      }
                    ],
                    "loc": {
                      "start": 5962,
                      "end": 6498
                    }
                  },
                  "loc": {
                    "start": 5949,
                    "end": 6498
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6507,
                      "end": 6519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6534,
                            "end": 6536
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6534,
                          "end": 6536
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6549,
                            "end": 6557
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6549,
                          "end": 6557
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6570,
                            "end": 6581
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6570,
                          "end": 6581
                        }
                      }
                    ],
                    "loc": {
                      "start": 6520,
                      "end": 6591
                    }
                  },
                  "loc": {
                    "start": 6507,
                    "end": 6591
                  }
                }
              ],
              "loc": {
                "start": 4559,
                "end": 6597
              }
            },
            "loc": {
              "start": 4541,
              "end": 6597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 6602,
                "end": 6616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6602,
              "end": 6616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 6621,
                "end": 6633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6621,
              "end": 6633
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6638,
                "end": 6641
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6652,
                      "end": 6661
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6652,
                    "end": 6661
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 6670,
                      "end": 6679
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6670,
                    "end": 6679
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6688,
                      "end": 6697
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6688,
                    "end": 6697
                  }
                }
              ],
              "loc": {
                "start": 6642,
                "end": 6703
              }
            },
            "loc": {
              "start": 6638,
              "end": 6703
            }
          }
        ],
        "loc": {
          "start": 3865,
          "end": 6705
        }
      },
      "loc": {
        "start": 3856,
        "end": 6705
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 6706,
          "end": 6717
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectVersion",
              "loc": {
                "start": 6724,
                "end": 6738
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6749,
                      "end": 6751
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6749,
                    "end": 6751
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6760,
                      "end": 6770
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6760,
                    "end": 6770
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6779,
                      "end": 6787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6779,
                    "end": 6787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6796,
                      "end": 6805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6796,
                    "end": 6805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6814,
                      "end": 6826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6814,
                    "end": 6826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6835,
                      "end": 6847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6835,
                    "end": 6847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6856,
                      "end": 6860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6875,
                            "end": 6877
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6875,
                          "end": 6877
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6890,
                            "end": 6899
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6890,
                          "end": 6899
                        }
                      }
                    ],
                    "loc": {
                      "start": 6861,
                      "end": 6909
                    }
                  },
                  "loc": {
                    "start": 6856,
                    "end": 6909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6918,
                      "end": 6930
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6945,
                            "end": 6947
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6945,
                          "end": 6947
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6960,
                            "end": 6968
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6960,
                          "end": 6968
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6981,
                            "end": 6992
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6981,
                          "end": 6992
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7005,
                            "end": 7009
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7005,
                          "end": 7009
                        }
                      }
                    ],
                    "loc": {
                      "start": 6931,
                      "end": 7019
                    }
                  },
                  "loc": {
                    "start": 6918,
                    "end": 7019
                  }
                }
              ],
              "loc": {
                "start": 6739,
                "end": 7025
              }
            },
            "loc": {
              "start": 6724,
              "end": 7025
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7030,
                "end": 7032
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7030,
              "end": 7032
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7037,
                "end": 7046
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7037,
              "end": 7046
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 7051,
                "end": 7070
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7051,
              "end": 7070
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 7075,
                "end": 7090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7075,
              "end": 7090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7095,
                "end": 7104
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7095,
              "end": 7104
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7109,
                "end": 7120
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7109,
              "end": 7120
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7125,
                "end": 7136
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7125,
              "end": 7136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7141,
                "end": 7145
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7141,
              "end": 7145
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7150,
                "end": 7156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7150,
              "end": 7156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7161,
                "end": 7171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7161,
              "end": 7171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7176,
                "end": 7188
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7202,
                      "end": 7218
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7199,
                    "end": 7218
                  }
                }
              ],
              "loc": {
                "start": 7189,
                "end": 7224
              }
            },
            "loc": {
              "start": 7176,
              "end": 7224
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 7229,
                "end": 7233
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 7247,
                      "end": 7255
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7244,
                    "end": 7255
                  }
                }
              ],
              "loc": {
                "start": 7234,
                "end": 7261
              }
            },
            "loc": {
              "start": 7229,
              "end": 7261
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7266,
                "end": 7269
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7280,
                      "end": 7289
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7280,
                    "end": 7289
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7298,
                      "end": 7307
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7298,
                    "end": 7307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7316,
                      "end": 7323
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7316,
                    "end": 7323
                  }
                }
              ],
              "loc": {
                "start": 7270,
                "end": 7329
              }
            },
            "loc": {
              "start": 7266,
              "end": 7329
            }
          }
        ],
        "loc": {
          "start": 6718,
          "end": 7331
        }
      },
      "loc": {
        "start": 6706,
        "end": 7331
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 7332,
          "end": 7343
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routineVersion",
              "loc": {
                "start": 7350,
                "end": 7364
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7375,
                      "end": 7377
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7375,
                    "end": 7377
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 7386,
                      "end": 7396
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7386,
                    "end": 7396
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 7405,
                      "end": 7418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7405,
                    "end": 7418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 7427,
                      "end": 7437
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7427,
                    "end": 7437
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 7446,
                      "end": 7455
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7446,
                    "end": 7455
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 7464,
                      "end": 7472
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7464,
                    "end": 7472
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7481,
                      "end": 7490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7481,
                    "end": 7490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 7499,
                      "end": 7503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 7518,
                            "end": 7520
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7518,
                          "end": 7520
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 7533,
                            "end": 7543
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7533,
                          "end": 7543
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7556,
                            "end": 7565
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7556,
                          "end": 7565
                        }
                      }
                    ],
                    "loc": {
                      "start": 7504,
                      "end": 7575
                    }
                  },
                  "loc": {
                    "start": 7499,
                    "end": 7575
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 7584,
                      "end": 7596
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 7611,
                            "end": 7613
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7611,
                          "end": 7613
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 7626,
                            "end": 7634
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7626,
                          "end": 7634
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7647,
                            "end": 7658
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7647,
                          "end": 7658
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 7671,
                            "end": 7683
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7671,
                          "end": 7683
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7696,
                            "end": 7700
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7696,
                          "end": 7700
                        }
                      }
                    ],
                    "loc": {
                      "start": 7597,
                      "end": 7710
                    }
                  },
                  "loc": {
                    "start": 7584,
                    "end": 7710
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 7719,
                      "end": 7731
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7719,
                    "end": 7731
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 7740,
                      "end": 7752
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7740,
                    "end": 7752
                  }
                }
              ],
              "loc": {
                "start": 7365,
                "end": 7758
              }
            },
            "loc": {
              "start": 7350,
              "end": 7758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7763,
                "end": 7765
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7763,
              "end": 7765
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7770,
                "end": 7779
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7770,
              "end": 7779
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 7784,
                "end": 7803
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7784,
              "end": 7803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 7808,
                "end": 7823
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7808,
              "end": 7823
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7828,
                "end": 7837
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7828,
              "end": 7837
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7842,
                "end": 7853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7842,
              "end": 7853
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7858,
                "end": 7869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7858,
              "end": 7869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7874,
                "end": 7878
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7874,
              "end": 7878
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7883,
                "end": 7889
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7883,
              "end": 7889
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7894,
                "end": 7904
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7894,
              "end": 7904
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 7909,
                "end": 7920
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7909,
              "end": 7920
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 7925,
                "end": 7944
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7925,
              "end": 7944
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7949,
                "end": 7961
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7975,
                      "end": 7991
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7972,
                    "end": 7991
                  }
                }
              ],
              "loc": {
                "start": 7962,
                "end": 7997
              }
            },
            "loc": {
              "start": 7949,
              "end": 7997
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 8002,
                "end": 8006
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 8020,
                      "end": 8028
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8017,
                    "end": 8028
                  }
                }
              ],
              "loc": {
                "start": 8007,
                "end": 8034
              }
            },
            "loc": {
              "start": 8002,
              "end": 8034
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8039,
                "end": 8042
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 8053,
                      "end": 8062
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8053,
                    "end": 8062
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 8071,
                      "end": 8080
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8071,
                    "end": 8080
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 8089,
                      "end": 8096
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8089,
                    "end": 8096
                  }
                }
              ],
              "loc": {
                "start": 8043,
                "end": 8102
              }
            },
            "loc": {
              "start": 8039,
              "end": 8102
            }
          }
        ],
        "loc": {
          "start": 7344,
          "end": 8104
        }
      },
      "loc": {
        "start": 7332,
        "end": 8104
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8105,
          "end": 8107
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8105,
        "end": 8107
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8108,
          "end": 8118
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8108,
        "end": 8118
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 8119,
          "end": 8129
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8119,
        "end": 8129
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 8130,
          "end": 8139
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8130,
        "end": 8139
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 8140,
          "end": 8147
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8140,
        "end": 8147
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 8148,
          "end": 8156
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8148,
        "end": 8156
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 8157,
          "end": 8167
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8174,
                "end": 8176
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8174,
              "end": 8176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 8181,
                "end": 8198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8181,
              "end": 8198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 8203,
                "end": 8215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8203,
              "end": 8215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 8220,
                "end": 8230
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8220,
              "end": 8230
            }
          }
        ],
        "loc": {
          "start": 8168,
          "end": 8232
        }
      },
      "loc": {
        "start": 8157,
        "end": 8232
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 8233,
          "end": 8244
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8251,
                "end": 8253
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8251,
              "end": 8253
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 8258,
                "end": 8272
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8258,
              "end": 8272
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 8277,
                "end": 8285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8277,
              "end": 8285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 8290,
                "end": 8299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8290,
              "end": 8299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 8304,
                "end": 8314
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8304,
              "end": 8314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 8319,
                "end": 8324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8319,
              "end": 8324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 8329,
                "end": 8336
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8329,
              "end": 8336
            }
          }
        ],
        "loc": {
          "start": 8245,
          "end": 8338
        }
      },
      "loc": {
        "start": 8233,
        "end": 8338
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8368,
          "end": 8370
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8368,
        "end": 8370
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8371,
          "end": 8381
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8371,
        "end": 8381
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 8382,
          "end": 8385
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8382,
        "end": 8385
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 8386,
          "end": 8395
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8386,
        "end": 8395
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 8396,
          "end": 8408
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8415,
                "end": 8417
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8415,
              "end": 8417
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 8422,
                "end": 8430
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8422,
              "end": 8430
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 8435,
                "end": 8446
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8435,
              "end": 8446
            }
          }
        ],
        "loc": {
          "start": 8409,
          "end": 8448
        }
      },
      "loc": {
        "start": 8396,
        "end": 8448
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8449,
          "end": 8452
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOwn",
              "loc": {
                "start": 8459,
                "end": 8464
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8459,
              "end": 8464
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8469,
                "end": 8481
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8469,
              "end": 8481
            }
          }
        ],
        "loc": {
          "start": 8453,
          "end": 8483
        }
      },
      "loc": {
        "start": 8449,
        "end": 8483
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8514,
          "end": 8516
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8514,
        "end": 8516
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8517,
          "end": 8527
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8517,
        "end": 8527
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 8528,
          "end": 8538
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8528,
        "end": 8538
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 8539,
          "end": 8550
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8539,
        "end": 8550
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8551,
          "end": 8557
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8551,
        "end": 8557
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8558,
          "end": 8563
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8558,
        "end": 8563
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8564,
          "end": 8568
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8564,
        "end": 8568
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8569,
          "end": 8581
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8569,
        "end": 8581
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 10,
          "end": 20
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 24,
            "end": 29
          }
        },
        "loc": {
          "start": 24,
          "end": 29
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 32,
                "end": 34
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 32,
              "end": 34
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 35,
                "end": 45
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 35,
              "end": 45
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 46,
                "end": 56
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 46,
              "end": 56
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 57,
                "end": 62
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 57,
              "end": 62
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 63,
              "end": 68
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 69,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 88,
                        "end": 100
                      }
                    },
                    "loc": {
                      "start": 88,
                      "end": 100
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 114,
                            "end": 130
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 111,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 101,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 81,
                    "end": 136
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 148,
                        "end": 152
                      }
                    },
                    "loc": {
                      "start": 148,
                      "end": 152
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 166,
                            "end": 174
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 163,
                          "end": 174
                        }
                      }
                    ],
                    "loc": {
                      "start": 153,
                      "end": 180
                    }
                  },
                  "loc": {
                    "start": 141,
                    "end": 180
                  }
                }
              ],
              "loc": {
                "start": 75,
                "end": 182
              }
            },
            "loc": {
              "start": 69,
              "end": 182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 183,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 193,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 207,
                      "end": 216
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 216
                  }
                }
              ],
              "loc": {
                "start": 187,
                "end": 218
              }
            },
            "loc": {
              "start": 183,
              "end": 218
            }
          }
        ],
        "loc": {
          "start": 30,
          "end": 220
        }
      },
      "loc": {
        "start": 1,
        "end": 220
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 230,
          "end": 239
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 243,
            "end": 247
          }
        },
        "loc": {
          "start": 243,
          "end": 247
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versions",
              "loc": {
                "start": 250,
                "end": 258
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 265,
                      "end": 277
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 288,
                            "end": 290
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 288,
                          "end": 290
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 299,
                            "end": 307
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 299,
                          "end": 307
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 316,
                            "end": 327
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 316,
                          "end": 327
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 336,
                            "end": 340
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 336,
                          "end": 340
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "pages",
                          "loc": {
                            "start": 349,
                            "end": 354
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 369,
                                  "end": 371
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 369,
                                "end": 371
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "pageIndex",
                                "loc": {
                                  "start": 384,
                                  "end": 393
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 384,
                                "end": 393
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "text",
                                "loc": {
                                  "start": 406,
                                  "end": 410
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 406,
                                "end": 410
                              }
                            }
                          ],
                          "loc": {
                            "start": 355,
                            "end": 420
                          }
                        },
                        "loc": {
                          "start": 349,
                          "end": 420
                        }
                      }
                    ],
                    "loc": {
                      "start": 278,
                      "end": 426
                    }
                  },
                  "loc": {
                    "start": 265,
                    "end": 426
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 431,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 431,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 438,
                      "end": 448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 438,
                    "end": 448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 453,
                      "end": 463
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 453,
                    "end": 463
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 468,
                      "end": 476
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 468,
                    "end": 476
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 481,
                      "end": 490
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 481,
                    "end": 490
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 495,
                      "end": 507
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 495,
                    "end": 507
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 512,
                      "end": 524
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 512,
                    "end": 524
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 529,
                      "end": 541
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 529,
                    "end": 541
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 546,
                      "end": 549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canComment",
                          "loc": {
                            "start": 560,
                            "end": 570
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 560,
                          "end": 570
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 579,
                            "end": 586
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 579,
                          "end": 586
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 595,
                            "end": 604
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 595,
                          "end": 604
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 613,
                            "end": 622
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 613,
                          "end": 622
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 631,
                            "end": 640
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 631,
                          "end": 640
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 649,
                            "end": 655
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 649,
                          "end": 655
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 664,
                            "end": 671
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 664,
                          "end": 671
                        }
                      }
                    ],
                    "loc": {
                      "start": 550,
                      "end": 677
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 677
                  }
                }
              ],
              "loc": {
                "start": 259,
                "end": 679
              }
            },
            "loc": {
              "start": 250,
              "end": 679
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 680,
                "end": 682
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 680,
              "end": 682
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 683,
                "end": 693
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 683,
              "end": 693
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 694,
                "end": 704
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 694,
              "end": 704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 705,
                "end": 714
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 705,
              "end": 714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 715,
                "end": 726
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 715,
              "end": 726
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 727,
                "end": 733
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 743,
                      "end": 753
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 740,
                    "end": 753
                  }
                }
              ],
              "loc": {
                "start": 734,
                "end": 755
              }
            },
            "loc": {
              "start": 727,
              "end": 755
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 756,
                "end": 761
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 775,
                        "end": 787
                      }
                    },
                    "loc": {
                      "start": 775,
                      "end": 787
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 801,
                            "end": 817
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 798,
                          "end": 817
                        }
                      }
                    ],
                    "loc": {
                      "start": 788,
                      "end": 823
                    }
                  },
                  "loc": {
                    "start": 768,
                    "end": 823
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 835,
                        "end": 839
                      }
                    },
                    "loc": {
                      "start": 835,
                      "end": 839
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 853,
                            "end": 861
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 850,
                          "end": 861
                        }
                      }
                    ],
                    "loc": {
                      "start": 840,
                      "end": 867
                    }
                  },
                  "loc": {
                    "start": 828,
                    "end": 867
                  }
                }
              ],
              "loc": {
                "start": 762,
                "end": 869
              }
            },
            "loc": {
              "start": 756,
              "end": 869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 870,
                "end": 881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 870,
              "end": 881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 882,
                "end": 896
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 882,
              "end": 896
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 897,
                "end": 902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 897,
              "end": 902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 903,
                "end": 912
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 903,
              "end": 912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 913,
                "end": 917
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Tag_list",
                    "loc": {
                      "start": 927,
                      "end": 935
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 924,
                    "end": 935
                  }
                }
              ],
              "loc": {
                "start": 918,
                "end": 937
              }
            },
            "loc": {
              "start": 913,
              "end": 937
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 938,
                "end": 952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 938,
              "end": 952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 953,
                "end": 958
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 953,
              "end": 958
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 959,
                "end": 962
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 969,
                      "end": 978
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 969,
                    "end": 978
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 983,
                      "end": 994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 983,
                    "end": 994
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 999,
                      "end": 1010
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 999,
                    "end": 1010
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1015,
                      "end": 1024
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1015,
                    "end": 1024
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1029,
                      "end": 1036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1029,
                    "end": 1036
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1041,
                      "end": 1049
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1041,
                    "end": 1049
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1054,
                      "end": 1066
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1054,
                    "end": 1066
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1071,
                      "end": 1079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1071,
                    "end": 1079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1084,
                      "end": 1092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1084,
                    "end": 1092
                  }
                }
              ],
              "loc": {
                "start": 963,
                "end": 1094
              }
            },
            "loc": {
              "start": 959,
              "end": 1094
            }
          }
        ],
        "loc": {
          "start": 248,
          "end": 1096
        }
      },
      "loc": {
        "start": 221,
        "end": 1096
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 1106,
          "end": 1122
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 1126,
            "end": 1138
          }
        },
        "loc": {
          "start": 1126,
          "end": 1138
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1141,
                "end": 1143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1141,
              "end": 1143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 1144,
                "end": 1155
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1144,
              "end": 1155
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 1156,
                "end": 1162
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1156,
              "end": 1162
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 1163,
                "end": 1175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1163,
              "end": 1175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1176,
                "end": 1179
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canAddMembers",
                    "loc": {
                      "start": 1186,
                      "end": 1199
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1186,
                    "end": 1199
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1204,
                      "end": 1213
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1204,
                    "end": 1213
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1218,
                      "end": 1229
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1218,
                    "end": 1229
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1234,
                      "end": 1243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1234,
                    "end": 1243
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1248,
                      "end": 1257
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1248,
                    "end": 1257
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1262,
                      "end": 1269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1262,
                    "end": 1269
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1274,
                      "end": 1286
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1274,
                    "end": 1286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1291,
                      "end": 1299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1291,
                    "end": 1299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 1304,
                      "end": 1318
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 1329,
                            "end": 1331
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1329,
                          "end": 1331
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 1340,
                            "end": 1350
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1340,
                          "end": 1350
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 1359,
                            "end": 1369
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1359,
                          "end": 1369
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 1378,
                            "end": 1385
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1378,
                          "end": 1385
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 1394,
                            "end": 1405
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1394,
                          "end": 1405
                        }
                      }
                    ],
                    "loc": {
                      "start": 1319,
                      "end": 1411
                    }
                  },
                  "loc": {
                    "start": 1304,
                    "end": 1411
                  }
                }
              ],
              "loc": {
                "start": 1180,
                "end": 1413
              }
            },
            "loc": {
              "start": 1176,
              "end": 1413
            }
          }
        ],
        "loc": {
          "start": 1139,
          "end": 1415
        }
      },
      "loc": {
        "start": 1097,
        "end": 1415
      }
    },
    "Reminder_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Reminder_full",
        "loc": {
          "start": 1425,
          "end": 1438
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Reminder",
          "loc": {
            "start": 1442,
            "end": 1450
          }
        },
        "loc": {
          "start": 1442,
          "end": 1450
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1453,
                "end": 1455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1453,
              "end": 1455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1456,
                "end": 1466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1456,
              "end": 1466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1467,
                "end": 1477
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1467,
              "end": 1477
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 1478,
                "end": 1482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1478,
              "end": 1482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 1483,
                "end": 1494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1483,
              "end": 1494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dueDate",
              "loc": {
                "start": 1495,
                "end": 1502
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1495,
              "end": 1502
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 1503,
                "end": 1508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1503,
              "end": 1508
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 1509,
                "end": 1519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1509,
              "end": 1519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderItems",
              "loc": {
                "start": 1520,
                "end": 1533
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1540,
                      "end": 1542
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1540,
                    "end": 1542
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1547,
                      "end": 1557
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1547,
                    "end": 1557
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1562,
                      "end": 1572
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1562,
                    "end": 1572
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1577,
                      "end": 1581
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1577,
                    "end": 1581
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1586,
                      "end": 1597
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1586,
                    "end": 1597
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dueDate",
                    "loc": {
                      "start": 1602,
                      "end": 1609
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1602,
                    "end": 1609
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "index",
                    "loc": {
                      "start": 1614,
                      "end": 1619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1614,
                    "end": 1619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 1624,
                      "end": 1634
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1624,
                    "end": 1634
                  }
                }
              ],
              "loc": {
                "start": 1534,
                "end": 1636
              }
            },
            "loc": {
              "start": 1520,
              "end": 1636
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminderList",
              "loc": {
                "start": 1637,
                "end": 1649
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1656,
                      "end": 1658
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1656,
                    "end": 1658
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1663,
                      "end": 1673
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1663,
                    "end": 1673
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1678,
                      "end": 1688
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1678,
                    "end": 1688
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusMode",
                    "loc": {
                      "start": 1693,
                      "end": 1702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 1713,
                            "end": 1719
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1734,
                                  "end": 1736
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1734,
                                "end": 1736
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 1749,
                                  "end": 1754
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1749,
                                "end": 1754
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 1767,
                                  "end": 1772
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1767,
                                "end": 1772
                              }
                            }
                          ],
                          "loc": {
                            "start": 1720,
                            "end": 1782
                          }
                        },
                        "loc": {
                          "start": 1713,
                          "end": 1782
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 1791,
                            "end": 1803
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1818,
                                  "end": 1820
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1818,
                                "end": 1820
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 1833,
                                  "end": 1843
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1833,
                                "end": 1843
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 1856,
                                  "end": 1868
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1887,
                                        "end": 1889
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1887,
                                      "end": 1889
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 1906,
                                        "end": 1914
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1906,
                                      "end": 1914
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 1931,
                                        "end": 1942
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1931,
                                      "end": 1942
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 1959,
                                        "end": 1963
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1959,
                                      "end": 1963
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1869,
                                  "end": 1977
                                }
                              },
                              "loc": {
                                "start": 1856,
                                "end": 1977
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 1990,
                                  "end": 1999
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2018,
                                        "end": 2020
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2018,
                                      "end": 2020
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 2037,
                                        "end": 2042
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2037,
                                      "end": 2042
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 2059,
                                        "end": 2063
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2059,
                                      "end": 2063
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 2080,
                                        "end": 2087
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2080,
                                      "end": 2087
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2104,
                                        "end": 2116
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2139,
                                              "end": 2141
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2139,
                                            "end": 2141
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2162,
                                              "end": 2170
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2162,
                                            "end": 2170
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2191,
                                              "end": 2202
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2191,
                                            "end": 2202
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2223,
                                              "end": 2227
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2223,
                                            "end": 2227
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2117,
                                        "end": 2245
                                      }
                                    },
                                    "loc": {
                                      "start": 2104,
                                      "end": 2245
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2000,
                                  "end": 2259
                                }
                              },
                              "loc": {
                                "start": 1990,
                                "end": 2259
                              }
                            }
                          ],
                          "loc": {
                            "start": 1804,
                            "end": 2269
                          }
                        },
                        "loc": {
                          "start": 1791,
                          "end": 2269
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 2278,
                            "end": 2286
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 2304,
                                  "end": 2319
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2301,
                                "end": 2319
                              }
                            }
                          ],
                          "loc": {
                            "start": 2287,
                            "end": 2329
                          }
                        },
                        "loc": {
                          "start": 2278,
                          "end": 2329
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2338,
                            "end": 2340
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2338,
                          "end": 2340
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2349,
                            "end": 2353
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2349,
                          "end": 2353
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2362,
                            "end": 2373
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2362,
                          "end": 2373
                        }
                      }
                    ],
                    "loc": {
                      "start": 1703,
                      "end": 2379
                    }
                  },
                  "loc": {
                    "start": 1693,
                    "end": 2379
                  }
                }
              ],
              "loc": {
                "start": 1650,
                "end": 2381
              }
            },
            "loc": {
              "start": 1637,
              "end": 2381
            }
          }
        ],
        "loc": {
          "start": 1451,
          "end": 2383
        }
      },
      "loc": {
        "start": 1416,
        "end": 2383
      }
    },
    "Resource_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Resource_list",
        "loc": {
          "start": 2393,
          "end": 2406
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Resource",
          "loc": {
            "start": 2410,
            "end": 2418
          }
        },
        "loc": {
          "start": 2410,
          "end": 2418
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2421,
                "end": 2423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2421,
              "end": 2423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "index",
              "loc": {
                "start": 2424,
                "end": 2429
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2424,
              "end": 2429
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "link",
              "loc": {
                "start": 2430,
                "end": 2434
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2430,
              "end": 2434
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "usedFor",
              "loc": {
                "start": 2435,
                "end": 2442
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2435,
              "end": 2442
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2443,
                "end": 2455
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2462,
                      "end": 2464
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2462,
                    "end": 2464
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2469,
                      "end": 2477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2469,
                    "end": 2477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2482,
                      "end": 2493
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2482,
                    "end": 2493
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2498,
                      "end": 2502
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2498,
                    "end": 2502
                  }
                }
              ],
              "loc": {
                "start": 2456,
                "end": 2504
              }
            },
            "loc": {
              "start": 2443,
              "end": 2504
            }
          }
        ],
        "loc": {
          "start": 2419,
          "end": 2506
        }
      },
      "loc": {
        "start": 2384,
        "end": 2506
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 2516,
          "end": 2531
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2535,
            "end": 2543
          }
        },
        "loc": {
          "start": 2535,
          "end": 2543
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2546,
                "end": 2548
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2546,
              "end": 2548
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2549,
                "end": 2559
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2549,
              "end": 2559
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2560,
                "end": 2570
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2560,
              "end": 2570
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 2571,
                "end": 2580
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2571,
              "end": 2580
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 2581,
                "end": 2588
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2581,
              "end": 2588
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 2589,
                "end": 2597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2589,
              "end": 2597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 2598,
                "end": 2608
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2615,
                      "end": 2617
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2615,
                    "end": 2617
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 2622,
                      "end": 2639
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2622,
                    "end": 2639
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 2644,
                      "end": 2656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2644,
                    "end": 2656
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 2661,
                      "end": 2671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2661,
                    "end": 2671
                  }
                }
              ],
              "loc": {
                "start": 2609,
                "end": 2673
              }
            },
            "loc": {
              "start": 2598,
              "end": 2673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 2674,
                "end": 2685
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2692,
                      "end": 2694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2692,
                    "end": 2694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 2699,
                      "end": 2713
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2699,
                    "end": 2713
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 2718,
                      "end": 2726
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2718,
                    "end": 2726
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 2731,
                      "end": 2740
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2731,
                    "end": 2740
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 2745,
                      "end": 2755
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2745,
                    "end": 2755
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 2760,
                      "end": 2765
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2760,
                    "end": 2765
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 2770,
                      "end": 2777
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2770,
                    "end": 2777
                  }
                }
              ],
              "loc": {
                "start": 2686,
                "end": 2779
              }
            },
            "loc": {
              "start": 2674,
              "end": 2779
            }
          }
        ],
        "loc": {
          "start": 2544,
          "end": 2781
        }
      },
      "loc": {
        "start": 2507,
        "end": 2781
      }
    },
    "Schedule_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_list",
        "loc": {
          "start": 2791,
          "end": 2804
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 2808,
            "end": 2816
          }
        },
        "loc": {
          "start": 2808,
          "end": 2816
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 2819,
                "end": 2825
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Label_list",
                    "loc": {
                      "start": 2835,
                      "end": 2845
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2832,
                    "end": 2845
                  }
                }
              ],
              "loc": {
                "start": 2826,
                "end": 2847
              }
            },
            "loc": {
              "start": 2819,
              "end": 2847
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 2848,
                "end": 2858
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 2865,
                      "end": 2871
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2882,
                            "end": 2884
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2882,
                          "end": 2884
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 2893,
                            "end": 2898
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2893,
                          "end": 2898
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2907,
                            "end": 2912
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2907,
                          "end": 2912
                        }
                      }
                    ],
                    "loc": {
                      "start": 2872,
                      "end": 2918
                    }
                  },
                  "loc": {
                    "start": 2865,
                    "end": 2918
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 2923,
                      "end": 2935
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2946,
                            "end": 2948
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2946,
                          "end": 2948
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2957,
                            "end": 2967
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2957,
                          "end": 2967
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2976,
                            "end": 2986
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2976,
                          "end": 2986
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 2995,
                            "end": 3004
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3019,
                                  "end": 3021
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3019,
                                "end": 3021
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3034,
                                  "end": 3044
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3034,
                                "end": 3044
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3057,
                                  "end": 3067
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3057,
                                "end": 3067
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3080,
                                  "end": 3084
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3080,
                                "end": 3084
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3097,
                                  "end": 3108
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3097,
                                "end": 3108
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 3121,
                                  "end": 3128
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3121,
                                "end": 3128
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 3141,
                                  "end": 3146
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3141,
                                "end": 3146
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 3159,
                                  "end": 3169
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3159,
                                "end": 3169
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 3182,
                                  "end": 3195
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3214,
                                        "end": 3216
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3214,
                                      "end": 3216
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3233,
                                        "end": 3243
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3233,
                                      "end": 3243
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3260,
                                        "end": 3270
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3260,
                                      "end": 3270
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3287,
                                        "end": 3291
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3287,
                                      "end": 3291
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3308,
                                        "end": 3319
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3308,
                                      "end": 3319
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 3336,
                                        "end": 3343
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3336,
                                      "end": 3343
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 3360,
                                        "end": 3365
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3360,
                                      "end": 3365
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 3382,
                                        "end": 3392
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3382,
                                      "end": 3392
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3196,
                                  "end": 3406
                                }
                              },
                              "loc": {
                                "start": 3182,
                                "end": 3406
                              }
                            }
                          ],
                          "loc": {
                            "start": 3005,
                            "end": 3416
                          }
                        },
                        "loc": {
                          "start": 2995,
                          "end": 3416
                        }
                      }
                    ],
                    "loc": {
                      "start": 2936,
                      "end": 3422
                    }
                  },
                  "loc": {
                    "start": 2923,
                    "end": 3422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 3427,
                      "end": 3439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3450,
                            "end": 3452
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3450,
                          "end": 3452
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3461,
                            "end": 3471
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3461,
                          "end": 3471
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 3480,
                            "end": 3492
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3507,
                                  "end": 3509
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3507,
                                "end": 3509
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 3522,
                                  "end": 3530
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3522,
                                "end": 3530
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3543,
                                  "end": 3554
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3543,
                                "end": 3554
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3567,
                                  "end": 3571
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3567,
                                "end": 3571
                              }
                            }
                          ],
                          "loc": {
                            "start": 3493,
                            "end": 3581
                          }
                        },
                        "loc": {
                          "start": 3480,
                          "end": 3581
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 3590,
                            "end": 3599
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3614,
                                  "end": 3616
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3614,
                                "end": 3616
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 3629,
                                  "end": 3634
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3629,
                                "end": 3634
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 3647,
                                  "end": 3651
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3647,
                                "end": 3651
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 3664,
                                  "end": 3671
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3664,
                                "end": 3671
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 3684,
                                  "end": 3696
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3715,
                                        "end": 3717
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3715,
                                      "end": 3717
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 3734,
                                        "end": 3742
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3734,
                                      "end": 3742
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3759,
                                        "end": 3770
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3759,
                                      "end": 3770
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3787,
                                        "end": 3791
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3787,
                                      "end": 3791
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3697,
                                  "end": 3805
                                }
                              },
                              "loc": {
                                "start": 3684,
                                "end": 3805
                              }
                            }
                          ],
                          "loc": {
                            "start": 3600,
                            "end": 3815
                          }
                        },
                        "loc": {
                          "start": 3590,
                          "end": 3815
                        }
                      }
                    ],
                    "loc": {
                      "start": 3440,
                      "end": 3821
                    }
                  },
                  "loc": {
                    "start": 3427,
                    "end": 3821
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3826,
                      "end": 3828
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3826,
                    "end": 3828
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3833,
                      "end": 3837
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3833,
                    "end": 3837
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3842,
                      "end": 3853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3842,
                    "end": 3853
                  }
                }
              ],
              "loc": {
                "start": 2859,
                "end": 3855
              }
            },
            "loc": {
              "start": 2848,
              "end": 3855
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetings",
              "loc": {
                "start": 3856,
                "end": 3864
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 3871,
                      "end": 3877
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Label_list",
                          "loc": {
                            "start": 3891,
                            "end": 3901
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3888,
                          "end": 3901
                        }
                      }
                    ],
                    "loc": {
                      "start": 3878,
                      "end": 3907
                    }
                  },
                  "loc": {
                    "start": 3871,
                    "end": 3907
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 3912,
                      "end": 3924
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3935,
                            "end": 3937
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3935,
                          "end": 3937
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3946,
                            "end": 3954
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3946,
                          "end": 3954
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3963,
                            "end": 3974
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3963,
                          "end": 3974
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "link",
                          "loc": {
                            "start": 3983,
                            "end": 3987
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3983,
                          "end": 3987
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3996,
                            "end": 4000
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3996,
                          "end": 4000
                        }
                      }
                    ],
                    "loc": {
                      "start": 3925,
                      "end": 4006
                    }
                  },
                  "loc": {
                    "start": 3912,
                    "end": 4006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4011,
                      "end": 4013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4011,
                    "end": 4013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 4018,
                      "end": 4040
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4018,
                    "end": 4040
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnOrganizationProfile",
                    "loc": {
                      "start": 4045,
                      "end": 4070
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4045,
                    "end": 4070
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 4075,
                      "end": 4087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4098,
                            "end": 4100
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4098,
                          "end": 4100
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 4109,
                            "end": 4120
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4109,
                          "end": 4120
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 4129,
                            "end": 4135
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4129,
                          "end": 4135
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 4144,
                            "end": 4156
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4144,
                          "end": 4156
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4165,
                            "end": 4168
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canAddMembers",
                                "loc": {
                                  "start": 4183,
                                  "end": 4196
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4183,
                                "end": 4196
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 4209,
                                  "end": 4218
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4209,
                                "end": 4218
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 4231,
                                  "end": 4242
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4231,
                                "end": 4242
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 4255,
                                  "end": 4264
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4255,
                                "end": 4264
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4277,
                                  "end": 4286
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4277,
                                "end": 4286
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 4299,
                                  "end": 4306
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4299,
                                "end": 4306
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 4319,
                                  "end": 4331
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4319,
                                "end": 4331
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 4344,
                                  "end": 4352
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4344,
                                "end": 4352
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 4365,
                                  "end": 4379
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4398,
                                        "end": 4400
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4398,
                                      "end": 4400
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4417,
                                        "end": 4427
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4417,
                                      "end": 4427
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4444,
                                        "end": 4454
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4444,
                                      "end": 4454
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 4471,
                                        "end": 4478
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4471,
                                      "end": 4478
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4495,
                                        "end": 4506
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4495,
                                      "end": 4506
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4380,
                                  "end": 4520
                                }
                              },
                              "loc": {
                                "start": 4365,
                                "end": 4520
                              }
                            }
                          ],
                          "loc": {
                            "start": 4169,
                            "end": 4530
                          }
                        },
                        "loc": {
                          "start": 4165,
                          "end": 4530
                        }
                      }
                    ],
                    "loc": {
                      "start": 4088,
                      "end": 4536
                    }
                  },
                  "loc": {
                    "start": 4075,
                    "end": 4536
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 4541,
                      "end": 4558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "members",
                          "loc": {
                            "start": 4569,
                            "end": 4576
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4591,
                                  "end": 4593
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4591,
                                "end": 4593
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4606,
                                  "end": 4616
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4606,
                                "end": 4616
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4629,
                                  "end": 4639
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4629,
                                "end": 4639
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 4652,
                                  "end": 4659
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4652,
                                "end": 4659
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4672,
                                  "end": 4683
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4672,
                                "end": 4683
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 4696,
                                  "end": 4701
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4720,
                                        "end": 4722
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4720,
                                      "end": 4722
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4739,
                                        "end": 4749
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4739,
                                      "end": 4749
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4766,
                                        "end": 4776
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4766,
                                      "end": 4776
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4793,
                                        "end": 4797
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4793,
                                      "end": 4797
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4814,
                                        "end": 4825
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4814,
                                      "end": 4825
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 4842,
                                        "end": 4854
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4842,
                                      "end": 4854
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 4871,
                                        "end": 4883
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4906,
                                              "end": 4908
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4906,
                                            "end": 4908
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 4929,
                                              "end": 4940
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4929,
                                            "end": 4940
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 4961,
                                              "end": 4967
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4961,
                                            "end": 4967
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 4988,
                                              "end": 5000
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4988,
                                            "end": 5000
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 5021,
                                              "end": 5024
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canAddMembers",
                                                  "loc": {
                                                    "start": 5051,
                                                    "end": 5064
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5051,
                                                  "end": 5064
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 5089,
                                                    "end": 5098
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5089,
                                                  "end": 5098
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 5123,
                                                    "end": 5134
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5123,
                                                  "end": 5134
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 5159,
                                                    "end": 5168
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5159,
                                                  "end": 5168
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 5193,
                                                    "end": 5202
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5193,
                                                  "end": 5202
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 5227,
                                                    "end": 5234
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5227,
                                                  "end": 5234
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 5259,
                                                    "end": 5271
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5259,
                                                  "end": 5271
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 5296,
                                                    "end": 5304
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5296,
                                                  "end": 5304
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 5329,
                                                    "end": 5343
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "selectionSet": {
                                                  "kind": "SelectionSet",
                                                  "selections": [
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "id",
                                                        "loc": {
                                                          "start": 5374,
                                                          "end": 5376
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5374,
                                                        "end": 5376
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 5405,
                                                          "end": 5415
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5405,
                                                        "end": 5415
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 5444,
                                                          "end": 5454
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5444,
                                                        "end": 5454
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 5483,
                                                          "end": 5490
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5483,
                                                        "end": 5490
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 5519,
                                                          "end": 5530
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5519,
                                                        "end": 5530
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 5344,
                                                    "end": 5556
                                                  }
                                                },
                                                "loc": {
                                                  "start": 5329,
                                                  "end": 5556
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5025,
                                              "end": 5578
                                            }
                                          },
                                          "loc": {
                                            "start": 5021,
                                            "end": 5578
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4884,
                                        "end": 5596
                                      }
                                    },
                                    "loc": {
                                      "start": 4871,
                                      "end": 5596
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5613,
                                        "end": 5625
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 5648,
                                              "end": 5650
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5648,
                                            "end": 5650
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5671,
                                              "end": 5679
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5671,
                                            "end": 5679
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5700,
                                              "end": 5711
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5700,
                                            "end": 5711
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5626,
                                        "end": 5729
                                      }
                                    },
                                    "loc": {
                                      "start": 5613,
                                      "end": 5729
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4702,
                                  "end": 5743
                                }
                              },
                              "loc": {
                                "start": 4696,
                                "end": 5743
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5756,
                                  "end": 5759
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 5778,
                                        "end": 5787
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5778,
                                      "end": 5787
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 5804,
                                        "end": 5813
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5804,
                                      "end": 5813
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5760,
                                  "end": 5827
                                }
                              },
                              "loc": {
                                "start": 5756,
                                "end": 5827
                              }
                            }
                          ],
                          "loc": {
                            "start": 4577,
                            "end": 5837
                          }
                        },
                        "loc": {
                          "start": 4569,
                          "end": 5837
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5846,
                            "end": 5848
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5846,
                          "end": 5848
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5857,
                            "end": 5867
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5857,
                          "end": 5867
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5876,
                            "end": 5886
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5876,
                          "end": 5886
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5895,
                            "end": 5899
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5895,
                          "end": 5899
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 5908,
                            "end": 5919
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5908,
                          "end": 5919
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 5928,
                            "end": 5940
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5928,
                          "end": 5940
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 5949,
                            "end": 5961
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 5976,
                                  "end": 5978
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5976,
                                "end": 5978
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 5991,
                                  "end": 6002
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5991,
                                "end": 6002
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 6015,
                                  "end": 6021
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6015,
                                "end": 6021
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 6034,
                                  "end": 6046
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6034,
                                "end": 6046
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 6059,
                                  "end": 6062
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canAddMembers",
                                      "loc": {
                                        "start": 6081,
                                        "end": 6094
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6081,
                                      "end": 6094
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 6111,
                                        "end": 6120
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6111,
                                      "end": 6120
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 6137,
                                        "end": 6148
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6137,
                                      "end": 6148
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 6165,
                                        "end": 6174
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6165,
                                      "end": 6174
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 6191,
                                        "end": 6200
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6191,
                                      "end": 6200
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 6217,
                                        "end": 6224
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6217,
                                      "end": 6224
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 6241,
                                        "end": 6253
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6241,
                                      "end": 6253
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 6270,
                                        "end": 6278
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6270,
                                      "end": 6278
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 6295,
                                        "end": 6309
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 6332,
                                              "end": 6334
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6332,
                                            "end": 6334
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6355,
                                              "end": 6365
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6355,
                                            "end": 6365
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6386,
                                              "end": 6396
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6386,
                                            "end": 6396
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 6417,
                                              "end": 6424
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6417,
                                            "end": 6424
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 6445,
                                              "end": 6456
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6445,
                                            "end": 6456
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6310,
                                        "end": 6474
                                      }
                                    },
                                    "loc": {
                                      "start": 6295,
                                      "end": 6474
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6063,
                                  "end": 6488
                                }
                              },
                              "loc": {
                                "start": 6059,
                                "end": 6488
                              }
                            }
                          ],
                          "loc": {
                            "start": 5962,
                            "end": 6498
                          }
                        },
                        "loc": {
                          "start": 5949,
                          "end": 6498
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6507,
                            "end": 6519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6534,
                                  "end": 6536
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6534,
                                "end": 6536
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6549,
                                  "end": 6557
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6549,
                                "end": 6557
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6570,
                                  "end": 6581
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6570,
                                "end": 6581
                              }
                            }
                          ],
                          "loc": {
                            "start": 6520,
                            "end": 6591
                          }
                        },
                        "loc": {
                          "start": 6507,
                          "end": 6591
                        }
                      }
                    ],
                    "loc": {
                      "start": 4559,
                      "end": 6597
                    }
                  },
                  "loc": {
                    "start": 4541,
                    "end": 6597
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 6602,
                      "end": 6616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6602,
                    "end": 6616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 6621,
                      "end": 6633
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6621,
                    "end": 6633
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6638,
                      "end": 6641
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6652,
                            "end": 6661
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6652,
                          "end": 6661
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 6670,
                            "end": 6679
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6670,
                          "end": 6679
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6688,
                            "end": 6697
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6688,
                          "end": 6697
                        }
                      }
                    ],
                    "loc": {
                      "start": 6642,
                      "end": 6703
                    }
                  },
                  "loc": {
                    "start": 6638,
                    "end": 6703
                  }
                }
              ],
              "loc": {
                "start": 3865,
                "end": 6705
              }
            },
            "loc": {
              "start": 3856,
              "end": 6705
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 6706,
                "end": 6717
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectVersion",
                    "loc": {
                      "start": 6724,
                      "end": 6738
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 6749,
                            "end": 6751
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6749,
                          "end": 6751
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6760,
                            "end": 6770
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6760,
                          "end": 6770
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6779,
                            "end": 6787
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6779,
                          "end": 6787
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6796,
                            "end": 6805
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6796,
                          "end": 6805
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6814,
                            "end": 6826
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6814,
                          "end": 6826
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6835,
                            "end": 6847
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6835,
                          "end": 6847
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6856,
                            "end": 6860
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6875,
                                  "end": 6877
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6875,
                                "end": 6877
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6890,
                                  "end": 6899
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6890,
                                "end": 6899
                              }
                            }
                          ],
                          "loc": {
                            "start": 6861,
                            "end": 6909
                          }
                        },
                        "loc": {
                          "start": 6856,
                          "end": 6909
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6918,
                            "end": 6930
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6945,
                                  "end": 6947
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6945,
                                "end": 6947
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6960,
                                  "end": 6968
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6960,
                                "end": 6968
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6981,
                                  "end": 6992
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6981,
                                "end": 6992
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7005,
                                  "end": 7009
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7005,
                                "end": 7009
                              }
                            }
                          ],
                          "loc": {
                            "start": 6931,
                            "end": 7019
                          }
                        },
                        "loc": {
                          "start": 6918,
                          "end": 7019
                        }
                      }
                    ],
                    "loc": {
                      "start": 6739,
                      "end": 7025
                    }
                  },
                  "loc": {
                    "start": 6724,
                    "end": 7025
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7030,
                      "end": 7032
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7030,
                    "end": 7032
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7037,
                      "end": 7046
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7037,
                    "end": 7046
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 7051,
                      "end": 7070
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7051,
                    "end": 7070
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 7075,
                      "end": 7090
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7075,
                    "end": 7090
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7095,
                      "end": 7104
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7095,
                    "end": 7104
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7109,
                      "end": 7120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7109,
                    "end": 7120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7125,
                      "end": 7136
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7125,
                    "end": 7136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7141,
                      "end": 7145
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7141,
                    "end": 7145
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7150,
                      "end": 7156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7150,
                    "end": 7156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7161,
                      "end": 7171
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7161,
                    "end": 7171
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7176,
                      "end": 7188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7202,
                            "end": 7218
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7199,
                          "end": 7218
                        }
                      }
                    ],
                    "loc": {
                      "start": 7189,
                      "end": 7224
                    }
                  },
                  "loc": {
                    "start": 7176,
                    "end": 7224
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 7229,
                      "end": 7233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 7247,
                            "end": 7255
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7244,
                          "end": 7255
                        }
                      }
                    ],
                    "loc": {
                      "start": 7234,
                      "end": 7261
                    }
                  },
                  "loc": {
                    "start": 7229,
                    "end": 7261
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7266,
                      "end": 7269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 7280,
                            "end": 7289
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7280,
                          "end": 7289
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7298,
                            "end": 7307
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7298,
                          "end": 7307
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7316,
                            "end": 7323
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7316,
                          "end": 7323
                        }
                      }
                    ],
                    "loc": {
                      "start": 7270,
                      "end": 7329
                    }
                  },
                  "loc": {
                    "start": 7266,
                    "end": 7329
                  }
                }
              ],
              "loc": {
                "start": 6718,
                "end": 7331
              }
            },
            "loc": {
              "start": 6706,
              "end": 7331
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 7332,
                "end": 7343
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routineVersion",
                    "loc": {
                      "start": 7350,
                      "end": 7364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 7375,
                            "end": 7377
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7375,
                          "end": 7377
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 7386,
                            "end": 7396
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7386,
                          "end": 7396
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 7405,
                            "end": 7418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7405,
                          "end": 7418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 7427,
                            "end": 7437
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7427,
                          "end": 7437
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 7446,
                            "end": 7455
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7446,
                          "end": 7455
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 7464,
                            "end": 7472
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7464,
                          "end": 7472
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7481,
                            "end": 7490
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7481,
                          "end": 7490
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 7499,
                            "end": 7503
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7518,
                                  "end": 7520
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7518,
                                "end": 7520
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 7533,
                                  "end": 7543
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7533,
                                "end": 7543
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 7556,
                                  "end": 7565
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7556,
                                "end": 7565
                              }
                            }
                          ],
                          "loc": {
                            "start": 7504,
                            "end": 7575
                          }
                        },
                        "loc": {
                          "start": 7499,
                          "end": 7575
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 7584,
                            "end": 7596
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 7611,
                                  "end": 7613
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7611,
                                "end": 7613
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 7626,
                                  "end": 7634
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7626,
                                "end": 7634
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7647,
                                  "end": 7658
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7647,
                                "end": 7658
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 7671,
                                  "end": 7683
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7671,
                                "end": 7683
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7696,
                                  "end": 7700
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7696,
                                "end": 7700
                              }
                            }
                          ],
                          "loc": {
                            "start": 7597,
                            "end": 7710
                          }
                        },
                        "loc": {
                          "start": 7584,
                          "end": 7710
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 7719,
                            "end": 7731
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7719,
                          "end": 7731
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 7740,
                            "end": 7752
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7740,
                          "end": 7752
                        }
                      }
                    ],
                    "loc": {
                      "start": 7365,
                      "end": 7758
                    }
                  },
                  "loc": {
                    "start": 7350,
                    "end": 7758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7763,
                      "end": 7765
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7763,
                    "end": 7765
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7770,
                      "end": 7779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7770,
                    "end": 7779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 7784,
                      "end": 7803
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7784,
                    "end": 7803
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 7808,
                      "end": 7823
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7808,
                    "end": 7823
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7828,
                      "end": 7837
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7828,
                    "end": 7837
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7842,
                      "end": 7853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7842,
                    "end": 7853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7858,
                      "end": 7869
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7858,
                    "end": 7869
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7874,
                      "end": 7878
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7874,
                    "end": 7878
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7883,
                      "end": 7889
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7883,
                    "end": 7889
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7894,
                      "end": 7904
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7894,
                    "end": 7904
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 7909,
                      "end": 7920
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7909,
                    "end": 7920
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 7925,
                      "end": 7944
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7925,
                    "end": 7944
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7949,
                      "end": 7961
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7975,
                            "end": 7991
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7972,
                          "end": 7991
                        }
                      }
                    ],
                    "loc": {
                      "start": 7962,
                      "end": 7997
                    }
                  },
                  "loc": {
                    "start": 7949,
                    "end": 7997
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 8002,
                      "end": 8006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 8020,
                            "end": 8028
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 8017,
                          "end": 8028
                        }
                      }
                    ],
                    "loc": {
                      "start": 8007,
                      "end": 8034
                    }
                  },
                  "loc": {
                    "start": 8002,
                    "end": 8034
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 8039,
                      "end": 8042
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 8053,
                            "end": 8062
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8053,
                          "end": 8062
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 8071,
                            "end": 8080
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8071,
                          "end": 8080
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 8089,
                            "end": 8096
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8089,
                          "end": 8096
                        }
                      }
                    ],
                    "loc": {
                      "start": 8043,
                      "end": 8102
                    }
                  },
                  "loc": {
                    "start": 8039,
                    "end": 8102
                  }
                }
              ],
              "loc": {
                "start": 7344,
                "end": 8104
              }
            },
            "loc": {
              "start": 7332,
              "end": 8104
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8105,
                "end": 8107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8105,
              "end": 8107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8108,
                "end": 8118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8108,
              "end": 8118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 8119,
                "end": 8129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8119,
              "end": 8129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 8130,
                "end": 8139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8130,
              "end": 8139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 8140,
                "end": 8147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8140,
              "end": 8147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 8148,
                "end": 8156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8148,
              "end": 8156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 8157,
                "end": 8167
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8174,
                      "end": 8176
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8174,
                    "end": 8176
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 8181,
                      "end": 8198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8181,
                    "end": 8198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 8203,
                      "end": 8215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8203,
                    "end": 8215
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 8220,
                      "end": 8230
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8220,
                    "end": 8230
                  }
                }
              ],
              "loc": {
                "start": 8168,
                "end": 8232
              }
            },
            "loc": {
              "start": 8157,
              "end": 8232
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 8233,
                "end": 8244
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8251,
                      "end": 8253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8251,
                    "end": 8253
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 8258,
                      "end": 8272
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8258,
                    "end": 8272
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 8277,
                      "end": 8285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8277,
                    "end": 8285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 8290,
                      "end": 8299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8290,
                    "end": 8299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 8304,
                      "end": 8314
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8304,
                    "end": 8314
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 8319,
                      "end": 8324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8319,
                    "end": 8324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 8329,
                      "end": 8336
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8329,
                    "end": 8336
                  }
                }
              ],
              "loc": {
                "start": 8245,
                "end": 8338
              }
            },
            "loc": {
              "start": 8233,
              "end": 8338
            }
          }
        ],
        "loc": {
          "start": 2817,
          "end": 8340
        }
      },
      "loc": {
        "start": 2782,
        "end": 8340
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 8350,
          "end": 8358
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 8362,
            "end": 8365
          }
        },
        "loc": {
          "start": 8362,
          "end": 8365
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8368,
                "end": 8370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8368,
              "end": 8370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8371,
                "end": 8381
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8371,
              "end": 8381
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 8382,
                "end": 8385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8382,
              "end": 8385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 8386,
                "end": 8395
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8386,
              "end": 8395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 8396,
                "end": 8408
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8415,
                      "end": 8417
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8415,
                    "end": 8417
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 8422,
                      "end": 8430
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8422,
                    "end": 8430
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 8435,
                      "end": 8446
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8435,
                    "end": 8446
                  }
                }
              ],
              "loc": {
                "start": 8409,
                "end": 8448
              }
            },
            "loc": {
              "start": 8396,
              "end": 8448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8449,
                "end": 8452
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isOwn",
                    "loc": {
                      "start": 8459,
                      "end": 8464
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8459,
                    "end": 8464
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8469,
                      "end": 8481
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8469,
                    "end": 8481
                  }
                }
              ],
              "loc": {
                "start": 8453,
                "end": 8483
              }
            },
            "loc": {
              "start": 8449,
              "end": 8483
            }
          }
        ],
        "loc": {
          "start": 8366,
          "end": 8485
        }
      },
      "loc": {
        "start": 8341,
        "end": 8485
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 8495,
          "end": 8503
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8507,
            "end": 8511
          }
        },
        "loc": {
          "start": 8507,
          "end": 8511
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8514,
                "end": 8516
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8514,
              "end": 8516
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8517,
                "end": 8527
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8517,
              "end": 8527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 8528,
                "end": 8538
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8528,
              "end": 8538
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 8539,
                "end": 8550
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8539,
              "end": 8550
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8551,
                "end": 8557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8551,
              "end": 8557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8558,
                "end": 8563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8558,
              "end": 8563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8564,
                "end": 8568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8564,
              "end": 8568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8569,
                "end": 8581
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8569,
              "end": 8581
            }
          }
        ],
        "loc": {
          "start": 8512,
          "end": 8583
        }
      },
      "loc": {
        "start": 8486,
        "end": 8583
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "home",
      "loc": {
        "start": 8591,
        "end": 8595
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8597,
              "end": 8602
            }
          },
          "loc": {
            "start": 8596,
            "end": 8602
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "HomeInput",
              "loc": {
                "start": 8604,
                "end": 8613
              }
            },
            "loc": {
              "start": 8604,
              "end": 8613
            }
          },
          "loc": {
            "start": 8604,
            "end": 8614
          }
        },
        "directives": [],
        "loc": {
          "start": 8596,
          "end": 8614
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "home",
            "loc": {
              "start": 8620,
              "end": 8624
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8625,
                  "end": 8630
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8633,
                    "end": 8638
                  }
                },
                "loc": {
                  "start": 8632,
                  "end": 8638
                }
              },
              "loc": {
                "start": 8625,
                "end": 8638
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "notes",
                  "loc": {
                    "start": 8646,
                    "end": 8651
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Note_list",
                        "loc": {
                          "start": 8665,
                          "end": 8674
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8662,
                        "end": 8674
                      }
                    }
                  ],
                  "loc": {
                    "start": 8652,
                    "end": 8680
                  }
                },
                "loc": {
                  "start": 8646,
                  "end": 8680
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 8685,
                    "end": 8694
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Reminder_full",
                        "loc": {
                          "start": 8708,
                          "end": 8721
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8705,
                        "end": 8721
                      }
                    }
                  ],
                  "loc": {
                    "start": 8695,
                    "end": 8727
                  }
                },
                "loc": {
                  "start": 8685,
                  "end": 8727
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 8732,
                    "end": 8741
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Resource_list",
                        "loc": {
                          "start": 8755,
                          "end": 8768
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8752,
                        "end": 8768
                      }
                    }
                  ],
                  "loc": {
                    "start": 8742,
                    "end": 8774
                  }
                },
                "loc": {
                  "start": 8732,
                  "end": 8774
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 8779,
                    "end": 8788
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Schedule_list",
                        "loc": {
                          "start": 8802,
                          "end": 8815
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8799,
                        "end": 8815
                      }
                    }
                  ],
                  "loc": {
                    "start": 8789,
                    "end": 8821
                  }
                },
                "loc": {
                  "start": 8779,
                  "end": 8821
                }
              }
            ],
            "loc": {
              "start": 8640,
              "end": 8825
            }
          },
          "loc": {
            "start": 8620,
            "end": 8825
          }
        }
      ],
      "loc": {
        "start": 8616,
        "end": 8827
      }
    },
    "loc": {
      "start": 8585,
      "end": 8827
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
