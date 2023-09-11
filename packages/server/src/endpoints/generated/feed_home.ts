export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 8650,
          "end": 8654
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8655,
              "end": 8660
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8663,
                "end": 8668
              }
            },
            "loc": {
              "start": 8662,
              "end": 8668
            }
          },
          "loc": {
            "start": 8655,
            "end": 8668
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
                "start": 8676,
                "end": 8681
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
                      "start": 8695,
                      "end": 8704
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8692,
                    "end": 8704
                  }
                }
              ],
              "loc": {
                "start": 8682,
                "end": 8710
              }
            },
            "loc": {
              "start": 8676,
              "end": 8710
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 8715,
                "end": 8724
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
                      "start": 8738,
                      "end": 8751
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8735,
                    "end": 8751
                  }
                }
              ],
              "loc": {
                "start": 8725,
                "end": 8757
              }
            },
            "loc": {
              "start": 8715,
              "end": 8757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 8762,
                "end": 8771
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
                      "start": 8785,
                      "end": 8798
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8782,
                    "end": 8798
                  }
                }
              ],
              "loc": {
                "start": 8772,
                "end": 8804
              }
            },
            "loc": {
              "start": 8762,
              "end": 8804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 8809,
                "end": 8818
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
                      "start": 8832,
                      "end": 8845
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8829,
                    "end": 8845
                  }
                }
              ],
              "loc": {
                "start": 8819,
                "end": 8851
              }
            },
            "loc": {
              "start": 8809,
              "end": 8851
            }
          }
        ],
        "loc": {
          "start": 8670,
          "end": 8855
        }
      },
      "loc": {
        "start": 8650,
        "end": 8855
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
              "value": "created_at",
              "loc": {
                "start": 4018,
                "end": 4028
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4018,
              "end": 4028
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4033,
                "end": 4043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4033,
              "end": 4043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "openToAnyoneWithInvite",
              "loc": {
                "start": 4048,
                "end": 4070
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4048,
              "end": 4070
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "showOnOrganizationProfile",
              "loc": {
                "start": 4075,
                "end": 4100
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4075,
              "end": 4100
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 4105,
                "end": 4117
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
                      "start": 4128,
                      "end": 4130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4128,
                    "end": 4130
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 4139,
                      "end": 4150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4139,
                    "end": 4150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 4159,
                      "end": 4165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4159,
                    "end": 4165
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 4174,
                      "end": 4186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4174,
                    "end": 4186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 4195,
                      "end": 4198
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
                            "start": 4213,
                            "end": 4226
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4213,
                          "end": 4226
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 4239,
                            "end": 4248
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4239,
                          "end": 4248
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canBookmark",
                          "loc": {
                            "start": 4261,
                            "end": 4272
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4261,
                          "end": 4272
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 4285,
                            "end": 4294
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4285,
                          "end": 4294
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 4307,
                            "end": 4316
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4307,
                          "end": 4316
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 4329,
                            "end": 4336
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4329,
                          "end": 4336
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isBookmarked",
                          "loc": {
                            "start": 4349,
                            "end": 4361
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4349,
                          "end": 4361
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isViewed",
                          "loc": {
                            "start": 4374,
                            "end": 4382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4374,
                          "end": 4382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "yourMembership",
                          "loc": {
                            "start": 4395,
                            "end": 4409
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
                                  "start": 4428,
                                  "end": 4430
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4428,
                                "end": 4430
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4447,
                                  "end": 4457
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4447,
                                "end": 4457
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4474,
                                  "end": 4484
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4474,
                                "end": 4484
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 4501,
                                  "end": 4508
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4501,
                                "end": 4508
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4525,
                                  "end": 4536
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4525,
                                "end": 4536
                              }
                            }
                          ],
                          "loc": {
                            "start": 4410,
                            "end": 4550
                          }
                        },
                        "loc": {
                          "start": 4395,
                          "end": 4550
                        }
                      }
                    ],
                    "loc": {
                      "start": 4199,
                      "end": 4560
                    }
                  },
                  "loc": {
                    "start": 4195,
                    "end": 4560
                  }
                }
              ],
              "loc": {
                "start": 4118,
                "end": 4566
              }
            },
            "loc": {
              "start": 4105,
              "end": 4566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "restrictedToRoles",
              "loc": {
                "start": 4571,
                "end": 4588
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
                      "start": 4599,
                      "end": 4606
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
                            "start": 4621,
                            "end": 4623
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4621,
                          "end": 4623
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4636,
                            "end": 4646
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4636,
                          "end": 4646
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4659,
                            "end": 4669
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4659,
                          "end": 4669
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 4682,
                            "end": 4689
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4682,
                          "end": 4689
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 4702,
                            "end": 4713
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4702,
                          "end": 4713
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "roles",
                          "loc": {
                            "start": 4726,
                            "end": 4731
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
                                  "start": 4750,
                                  "end": 4752
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4750,
                                "end": 4752
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4769,
                                  "end": 4779
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4769,
                                "end": 4779
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4796,
                                  "end": 4806
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4796,
                                "end": 4806
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4823,
                                  "end": 4827
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4823,
                                "end": 4827
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4844,
                                  "end": 4855
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4844,
                                "end": 4855
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "membersCount",
                                "loc": {
                                  "start": 4872,
                                  "end": 4884
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4872,
                                "end": 4884
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "organization",
                                "loc": {
                                  "start": 4901,
                                  "end": 4913
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
                                        "start": 4936,
                                        "end": 4938
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4936,
                                      "end": 4938
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bannerImage",
                                      "loc": {
                                        "start": 4959,
                                        "end": 4970
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4959,
                                      "end": 4970
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "handle",
                                      "loc": {
                                        "start": 4991,
                                        "end": 4997
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4991,
                                      "end": 4997
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "profileImage",
                                      "loc": {
                                        "start": 5018,
                                        "end": 5030
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5018,
                                      "end": 5030
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5051,
                                        "end": 5054
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
                                              "start": 5081,
                                              "end": 5094
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5081,
                                            "end": 5094
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 5119,
                                              "end": 5128
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5119,
                                            "end": 5128
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canBookmark",
                                            "loc": {
                                              "start": 5153,
                                              "end": 5164
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5153,
                                            "end": 5164
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canReport",
                                            "loc": {
                                              "start": 5189,
                                              "end": 5198
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5189,
                                            "end": 5198
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 5223,
                                              "end": 5232
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5223,
                                            "end": 5232
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 5257,
                                              "end": 5264
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5257,
                                            "end": 5264
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5289,
                                              "end": 5301
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5289,
                                            "end": 5301
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isViewed",
                                            "loc": {
                                              "start": 5326,
                                              "end": 5334
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5326,
                                            "end": 5334
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "yourMembership",
                                            "loc": {
                                              "start": 5359,
                                              "end": 5373
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
                                                    "start": 5404,
                                                    "end": 5406
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5404,
                                                  "end": 5406
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 5435,
                                                    "end": 5445
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5435,
                                                  "end": 5445
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 5474,
                                                    "end": 5484
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5474,
                                                  "end": 5484
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isAdmin",
                                                  "loc": {
                                                    "start": 5513,
                                                    "end": 5520
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5513,
                                                  "end": 5520
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "permissions",
                                                  "loc": {
                                                    "start": 5549,
                                                    "end": 5560
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5549,
                                                  "end": 5560
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5374,
                                              "end": 5586
                                            }
                                          },
                                          "loc": {
                                            "start": 5359,
                                            "end": 5586
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5055,
                                        "end": 5608
                                      }
                                    },
                                    "loc": {
                                      "start": 5051,
                                      "end": 5608
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4914,
                                  "end": 5626
                                }
                              },
                              "loc": {
                                "start": 4901,
                                "end": 5626
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5643,
                                  "end": 5655
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
                                        "start": 5678,
                                        "end": 5680
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5678,
                                      "end": 5680
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5701,
                                        "end": 5709
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5701,
                                      "end": 5709
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5730,
                                        "end": 5741
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5730,
                                      "end": 5741
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5656,
                                  "end": 5759
                                }
                              },
                              "loc": {
                                "start": 5643,
                                "end": 5759
                              }
                            }
                          ],
                          "loc": {
                            "start": 4732,
                            "end": 5773
                          }
                        },
                        "loc": {
                          "start": 4726,
                          "end": 5773
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5786,
                            "end": 5789
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
                                  "start": 5808,
                                  "end": 5817
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5808,
                                "end": 5817
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 5834,
                                  "end": 5843
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5834,
                                "end": 5843
                              }
                            }
                          ],
                          "loc": {
                            "start": 5790,
                            "end": 5857
                          }
                        },
                        "loc": {
                          "start": 5786,
                          "end": 5857
                        }
                      }
                    ],
                    "loc": {
                      "start": 4607,
                      "end": 5867
                    }
                  },
                  "loc": {
                    "start": 4599,
                    "end": 5867
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5876,
                      "end": 5878
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5876,
                    "end": 5878
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5887,
                      "end": 5897
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5887,
                    "end": 5897
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5906,
                      "end": 5916
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5906,
                    "end": 5916
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5925,
                      "end": 5929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5925,
                    "end": 5929
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 5938,
                      "end": 5949
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5938,
                    "end": 5949
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 5958,
                      "end": 5970
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5958,
                    "end": 5970
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5979,
                      "end": 5991
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
                            "start": 6006,
                            "end": 6008
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6006,
                          "end": 6008
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 6021,
                            "end": 6032
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6021,
                          "end": 6032
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 6045,
                            "end": 6051
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6045,
                          "end": 6051
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 6064,
                            "end": 6076
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6064,
                          "end": 6076
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 6089,
                            "end": 6092
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
                                  "start": 6111,
                                  "end": 6124
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6111,
                                "end": 6124
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 6141,
                                  "end": 6150
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6141,
                                "end": 6150
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 6167,
                                  "end": 6178
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6167,
                                "end": 6178
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 6195,
                                  "end": 6204
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6195,
                                "end": 6204
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 6221,
                                  "end": 6230
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6221,
                                "end": 6230
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 6247,
                                  "end": 6254
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6247,
                                "end": 6254
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 6271,
                                  "end": 6283
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6271,
                                "end": 6283
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 6300,
                                  "end": 6308
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6300,
                                "end": 6308
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 6325,
                                  "end": 6339
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
                                        "start": 6362,
                                        "end": 6364
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6362,
                                      "end": 6364
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 6385,
                                        "end": 6395
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6385,
                                      "end": 6395
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 6416,
                                        "end": 6426
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6416,
                                      "end": 6426
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 6447,
                                        "end": 6454
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6447,
                                      "end": 6454
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 6475,
                                        "end": 6486
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6475,
                                      "end": 6486
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6340,
                                  "end": 6504
                                }
                              },
                              "loc": {
                                "start": 6325,
                                "end": 6504
                              }
                            }
                          ],
                          "loc": {
                            "start": 6093,
                            "end": 6518
                          }
                        },
                        "loc": {
                          "start": 6089,
                          "end": 6518
                        }
                      }
                    ],
                    "loc": {
                      "start": 5992,
                      "end": 6528
                    }
                  },
                  "loc": {
                    "start": 5979,
                    "end": 6528
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6537,
                      "end": 6549
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
                            "start": 6564,
                            "end": 6566
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6564,
                          "end": 6566
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6579,
                            "end": 6587
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6579,
                          "end": 6587
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6600,
                            "end": 6611
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6600,
                          "end": 6611
                        }
                      }
                    ],
                    "loc": {
                      "start": 6550,
                      "end": 6621
                    }
                  },
                  "loc": {
                    "start": 6537,
                    "end": 6621
                  }
                }
              ],
              "loc": {
                "start": 4589,
                "end": 6627
              }
            },
            "loc": {
              "start": 4571,
              "end": 6627
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 6632,
                "end": 6646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6632,
              "end": 6646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 6651,
                "end": 6663
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6651,
              "end": 6663
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6668,
                "end": 6671
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
                      "start": 6682,
                      "end": 6691
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6682,
                    "end": 6691
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 6700,
                      "end": 6709
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6700,
                    "end": 6709
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6718,
                      "end": 6727
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6718,
                    "end": 6727
                  }
                }
              ],
              "loc": {
                "start": 6672,
                "end": 6733
              }
            },
            "loc": {
              "start": 6668,
              "end": 6733
            }
          }
        ],
        "loc": {
          "start": 3865,
          "end": 6735
        }
      },
      "loc": {
        "start": 3856,
        "end": 6735
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 6736,
          "end": 6747
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
                "start": 6754,
                "end": 6768
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
                      "start": 6779,
                      "end": 6781
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6779,
                    "end": 6781
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6790,
                      "end": 6800
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6790,
                    "end": 6800
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6809,
                      "end": 6817
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6809,
                    "end": 6817
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6826,
                      "end": 6835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6826,
                    "end": 6835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6844,
                      "end": 6856
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6844,
                    "end": 6856
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6865,
                      "end": 6877
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6865,
                    "end": 6877
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6886,
                      "end": 6890
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
                            "start": 6905,
                            "end": 6907
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6905,
                          "end": 6907
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6920,
                            "end": 6929
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6920,
                          "end": 6929
                        }
                      }
                    ],
                    "loc": {
                      "start": 6891,
                      "end": 6939
                    }
                  },
                  "loc": {
                    "start": 6886,
                    "end": 6939
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6948,
                      "end": 6960
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
                            "start": 6975,
                            "end": 6977
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6975,
                          "end": 6977
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6990,
                            "end": 6998
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6990,
                          "end": 6998
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7011,
                            "end": 7022
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7011,
                          "end": 7022
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7035,
                            "end": 7039
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7035,
                          "end": 7039
                        }
                      }
                    ],
                    "loc": {
                      "start": 6961,
                      "end": 7049
                    }
                  },
                  "loc": {
                    "start": 6948,
                    "end": 7049
                  }
                }
              ],
              "loc": {
                "start": 6769,
                "end": 7055
              }
            },
            "loc": {
              "start": 6754,
              "end": 7055
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7060,
                "end": 7062
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7060,
              "end": 7062
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7067,
                "end": 7076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7067,
              "end": 7076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 7081,
                "end": 7100
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7081,
              "end": 7100
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 7105,
                "end": 7120
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7105,
              "end": 7120
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7125,
                "end": 7134
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7125,
              "end": 7134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7139,
                "end": 7150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7139,
              "end": 7150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7155,
                "end": 7166
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7155,
              "end": 7166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7171,
                "end": 7175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7171,
              "end": 7175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7180,
                "end": 7186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7180,
              "end": 7186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7191,
                "end": 7201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7191,
              "end": 7201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7206,
                "end": 7218
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
                      "start": 7232,
                      "end": 7248
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7229,
                    "end": 7248
                  }
                }
              ],
              "loc": {
                "start": 7219,
                "end": 7254
              }
            },
            "loc": {
              "start": 7206,
              "end": 7254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 7259,
                "end": 7263
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
                      "start": 7277,
                      "end": 7285
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7274,
                    "end": 7285
                  }
                }
              ],
              "loc": {
                "start": 7264,
                "end": 7291
              }
            },
            "loc": {
              "start": 7259,
              "end": 7291
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7296,
                "end": 7299
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
                      "start": 7310,
                      "end": 7319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7310,
                    "end": 7319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7328,
                      "end": 7337
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7328,
                    "end": 7337
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7346,
                      "end": 7353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7346,
                    "end": 7353
                  }
                }
              ],
              "loc": {
                "start": 7300,
                "end": 7359
              }
            },
            "loc": {
              "start": 7296,
              "end": 7359
            }
          }
        ],
        "loc": {
          "start": 6748,
          "end": 7361
        }
      },
      "loc": {
        "start": 6736,
        "end": 7361
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 7362,
          "end": 7373
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
                "start": 7380,
                "end": 7394
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
                      "start": 7405,
                      "end": 7407
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7405,
                    "end": 7407
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 7416,
                      "end": 7426
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7416,
                    "end": 7426
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 7435,
                      "end": 7448
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7435,
                    "end": 7448
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 7457,
                      "end": 7467
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7457,
                    "end": 7467
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 7476,
                      "end": 7485
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7476,
                    "end": 7485
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 7494,
                      "end": 7502
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7494,
                    "end": 7502
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7511,
                      "end": 7520
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7511,
                    "end": 7520
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 7529,
                      "end": 7533
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
                            "start": 7548,
                            "end": 7550
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7548,
                          "end": 7550
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 7563,
                            "end": 7573
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7563,
                          "end": 7573
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7586,
                            "end": 7595
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7586,
                          "end": 7595
                        }
                      }
                    ],
                    "loc": {
                      "start": 7534,
                      "end": 7605
                    }
                  },
                  "loc": {
                    "start": 7529,
                    "end": 7605
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 7614,
                      "end": 7626
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
                            "start": 7641,
                            "end": 7643
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7641,
                          "end": 7643
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 7656,
                            "end": 7664
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7656,
                          "end": 7664
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7677,
                            "end": 7688
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7677,
                          "end": 7688
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 7701,
                            "end": 7713
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7701,
                          "end": 7713
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7726,
                            "end": 7730
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7726,
                          "end": 7730
                        }
                      }
                    ],
                    "loc": {
                      "start": 7627,
                      "end": 7740
                    }
                  },
                  "loc": {
                    "start": 7614,
                    "end": 7740
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 7749,
                      "end": 7761
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7749,
                    "end": 7761
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 7770,
                      "end": 7782
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7770,
                    "end": 7782
                  }
                }
              ],
              "loc": {
                "start": 7395,
                "end": 7788
              }
            },
            "loc": {
              "start": 7380,
              "end": 7788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7793,
                "end": 7795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7793,
              "end": 7795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7800,
                "end": 7809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7800,
              "end": 7809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 7814,
                "end": 7833
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7814,
              "end": 7833
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 7838,
                "end": 7853
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7838,
              "end": 7853
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7858,
                "end": 7867
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7858,
              "end": 7867
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7872,
                "end": 7883
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7872,
              "end": 7883
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7888,
                "end": 7899
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7888,
              "end": 7899
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7904,
                "end": 7908
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7904,
              "end": 7908
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7913,
                "end": 7919
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7913,
              "end": 7919
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7924,
                "end": 7934
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7924,
              "end": 7934
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 7939,
                "end": 7950
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7939,
              "end": 7950
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 7955,
                "end": 7974
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7955,
              "end": 7974
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7979,
                "end": 7991
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
                      "start": 8005,
                      "end": 8021
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8002,
                    "end": 8021
                  }
                }
              ],
              "loc": {
                "start": 7992,
                "end": 8027
              }
            },
            "loc": {
              "start": 7979,
              "end": 8027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 8032,
                "end": 8036
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
                      "start": 8050,
                      "end": 8058
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8047,
                    "end": 8058
                  }
                }
              ],
              "loc": {
                "start": 8037,
                "end": 8064
              }
            },
            "loc": {
              "start": 8032,
              "end": 8064
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8069,
                "end": 8072
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
                      "start": 8083,
                      "end": 8092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8083,
                    "end": 8092
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 8101,
                      "end": 8110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8101,
                    "end": 8110
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 8119,
                      "end": 8126
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8119,
                    "end": 8126
                  }
                }
              ],
              "loc": {
                "start": 8073,
                "end": 8132
              }
            },
            "loc": {
              "start": 8069,
              "end": 8132
            }
          }
        ],
        "loc": {
          "start": 7374,
          "end": 8134
        }
      },
      "loc": {
        "start": 7362,
        "end": 8134
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8135,
          "end": 8137
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8135,
        "end": 8137
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8138,
          "end": 8148
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8138,
        "end": 8148
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 8149,
          "end": 8159
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8149,
        "end": 8159
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 8160,
          "end": 8169
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8160,
        "end": 8169
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 8170,
          "end": 8177
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8170,
        "end": 8177
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 8178,
          "end": 8186
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8178,
        "end": 8186
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 8187,
          "end": 8197
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
                "start": 8204,
                "end": 8206
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8204,
              "end": 8206
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 8211,
                "end": 8228
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8211,
              "end": 8228
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 8233,
                "end": 8245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8233,
              "end": 8245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 8250,
                "end": 8260
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8250,
              "end": 8260
            }
          }
        ],
        "loc": {
          "start": 8198,
          "end": 8262
        }
      },
      "loc": {
        "start": 8187,
        "end": 8262
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 8263,
          "end": 8274
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
                "start": 8281,
                "end": 8283
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8281,
              "end": 8283
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 8288,
                "end": 8302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8288,
              "end": 8302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 8307,
                "end": 8315
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8307,
              "end": 8315
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 8320,
                "end": 8329
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8320,
              "end": 8329
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 8334,
                "end": 8344
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8334,
              "end": 8344
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 8349,
                "end": 8354
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8349,
              "end": 8354
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 8359,
                "end": 8366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8359,
              "end": 8366
            }
          }
        ],
        "loc": {
          "start": 8275,
          "end": 8368
        }
      },
      "loc": {
        "start": 8263,
        "end": 8368
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8398,
          "end": 8400
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8398,
        "end": 8400
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8401,
          "end": 8411
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8401,
        "end": 8411
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 8412,
          "end": 8415
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8412,
        "end": 8415
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 8416,
          "end": 8425
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8416,
        "end": 8425
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 8426,
          "end": 8438
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
                "start": 8445,
                "end": 8447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8445,
              "end": 8447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 8452,
                "end": 8460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8452,
              "end": 8460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 8465,
                "end": 8476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8465,
              "end": 8476
            }
          }
        ],
        "loc": {
          "start": 8439,
          "end": 8478
        }
      },
      "loc": {
        "start": 8426,
        "end": 8478
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8479,
          "end": 8482
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
                "start": 8489,
                "end": 8494
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8489,
              "end": 8494
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8499,
                "end": 8511
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8499,
              "end": 8511
            }
          }
        ],
        "loc": {
          "start": 8483,
          "end": 8513
        }
      },
      "loc": {
        "start": 8479,
        "end": 8513
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8544,
          "end": 8546
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8544,
        "end": 8546
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8547,
          "end": 8557
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8547,
        "end": 8557
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 8558,
          "end": 8568
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8558,
        "end": 8568
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 8569,
          "end": 8580
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8569,
        "end": 8580
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8581,
          "end": 8587
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8581,
        "end": 8587
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8588,
          "end": 8593
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8588,
        "end": 8593
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8594,
          "end": 8598
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8594,
        "end": 8598
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8599,
          "end": 8611
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8599,
        "end": 8611
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
                    "value": "created_at",
                    "loc": {
                      "start": 4018,
                      "end": 4028
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4018,
                    "end": 4028
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4033,
                      "end": 4043
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4033,
                    "end": 4043
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "openToAnyoneWithInvite",
                    "loc": {
                      "start": 4048,
                      "end": 4070
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4048,
                    "end": 4070
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "showOnOrganizationProfile",
                    "loc": {
                      "start": 4075,
                      "end": 4100
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4075,
                    "end": 4100
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 4105,
                      "end": 4117
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
                            "start": 4128,
                            "end": 4130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4128,
                          "end": 4130
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 4139,
                            "end": 4150
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4139,
                          "end": 4150
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 4159,
                            "end": 4165
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4159,
                          "end": 4165
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 4174,
                            "end": 4186
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4174,
                          "end": 4186
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4195,
                            "end": 4198
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
                                  "start": 4213,
                                  "end": 4226
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4213,
                                "end": 4226
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 4239,
                                  "end": 4248
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4239,
                                "end": 4248
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 4261,
                                  "end": 4272
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4261,
                                "end": 4272
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 4285,
                                  "end": 4294
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4285,
                                "end": 4294
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4307,
                                  "end": 4316
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4307,
                                "end": 4316
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 4329,
                                  "end": 4336
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4329,
                                "end": 4336
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 4349,
                                  "end": 4361
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4349,
                                "end": 4361
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 4374,
                                  "end": 4382
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4374,
                                "end": 4382
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 4395,
                                  "end": 4409
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
                                        "start": 4428,
                                        "end": 4430
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4428,
                                      "end": 4430
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4447,
                                        "end": 4457
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4447,
                                      "end": 4457
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4474,
                                        "end": 4484
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4474,
                                      "end": 4484
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 4501,
                                        "end": 4508
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4501,
                                      "end": 4508
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4525,
                                        "end": 4536
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4525,
                                      "end": 4536
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4410,
                                  "end": 4550
                                }
                              },
                              "loc": {
                                "start": 4395,
                                "end": 4550
                              }
                            }
                          ],
                          "loc": {
                            "start": 4199,
                            "end": 4560
                          }
                        },
                        "loc": {
                          "start": 4195,
                          "end": 4560
                        }
                      }
                    ],
                    "loc": {
                      "start": 4118,
                      "end": 4566
                    }
                  },
                  "loc": {
                    "start": 4105,
                    "end": 4566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "restrictedToRoles",
                    "loc": {
                      "start": 4571,
                      "end": 4588
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
                            "start": 4599,
                            "end": 4606
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
                                  "start": 4621,
                                  "end": 4623
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4621,
                                "end": 4623
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4636,
                                  "end": 4646
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4636,
                                "end": 4646
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 4659,
                                  "end": 4669
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4659,
                                "end": 4669
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isAdmin",
                                "loc": {
                                  "start": 4682,
                                  "end": 4689
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4682,
                                "end": 4689
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "permissions",
                                "loc": {
                                  "start": 4702,
                                  "end": 4713
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4702,
                                "end": 4713
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "roles",
                                "loc": {
                                  "start": 4726,
                                  "end": 4731
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
                                        "start": 4750,
                                        "end": 4752
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4750,
                                      "end": 4752
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4769,
                                        "end": 4779
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4769,
                                      "end": 4779
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4796,
                                        "end": 4806
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4796,
                                      "end": 4806
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4823,
                                        "end": 4827
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4823,
                                      "end": 4827
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 4844,
                                        "end": 4855
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4844,
                                      "end": 4855
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "membersCount",
                                      "loc": {
                                        "start": 4872,
                                        "end": 4884
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4872,
                                      "end": 4884
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "organization",
                                      "loc": {
                                        "start": 4901,
                                        "end": 4913
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
                                              "start": 4936,
                                              "end": 4938
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4936,
                                            "end": 4938
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bannerImage",
                                            "loc": {
                                              "start": 4959,
                                              "end": 4970
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4959,
                                            "end": 4970
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "handle",
                                            "loc": {
                                              "start": 4991,
                                              "end": 4997
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4991,
                                            "end": 4997
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "profileImage",
                                            "loc": {
                                              "start": 5018,
                                              "end": 5030
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5018,
                                            "end": 5030
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 5051,
                                              "end": 5054
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
                                                    "start": 5081,
                                                    "end": 5094
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5081,
                                                  "end": 5094
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 5119,
                                                    "end": 5128
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5119,
                                                  "end": 5128
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canBookmark",
                                                  "loc": {
                                                    "start": 5153,
                                                    "end": 5164
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5153,
                                                  "end": 5164
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canReport",
                                                  "loc": {
                                                    "start": 5189,
                                                    "end": 5198
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5189,
                                                  "end": 5198
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 5223,
                                                    "end": 5232
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5223,
                                                  "end": 5232
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 5257,
                                                    "end": 5264
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5257,
                                                  "end": 5264
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 5289,
                                                    "end": 5301
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5289,
                                                  "end": 5301
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isViewed",
                                                  "loc": {
                                                    "start": 5326,
                                                    "end": 5334
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5326,
                                                  "end": 5334
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "yourMembership",
                                                  "loc": {
                                                    "start": 5359,
                                                    "end": 5373
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
                                                          "start": 5404,
                                                          "end": 5406
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5404,
                                                        "end": 5406
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 5435,
                                                          "end": 5445
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5435,
                                                        "end": 5445
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 5474,
                                                          "end": 5484
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5474,
                                                        "end": 5484
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isAdmin",
                                                        "loc": {
                                                          "start": 5513,
                                                          "end": 5520
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5513,
                                                        "end": 5520
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "permissions",
                                                        "loc": {
                                                          "start": 5549,
                                                          "end": 5560
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5549,
                                                        "end": 5560
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 5374,
                                                    "end": 5586
                                                  }
                                                },
                                                "loc": {
                                                  "start": 5359,
                                                  "end": 5586
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5055,
                                              "end": 5608
                                            }
                                          },
                                          "loc": {
                                            "start": 5051,
                                            "end": 5608
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4914,
                                        "end": 5626
                                      }
                                    },
                                    "loc": {
                                      "start": 4901,
                                      "end": 5626
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5643,
                                        "end": 5655
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
                                              "start": 5678,
                                              "end": 5680
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5678,
                                            "end": 5680
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5701,
                                              "end": 5709
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5701,
                                            "end": 5709
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5730,
                                              "end": 5741
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5730,
                                            "end": 5741
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5656,
                                        "end": 5759
                                      }
                                    },
                                    "loc": {
                                      "start": 5643,
                                      "end": 5759
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4732,
                                  "end": 5773
                                }
                              },
                              "loc": {
                                "start": 4726,
                                "end": 5773
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5786,
                                  "end": 5789
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
                                        "start": 5808,
                                        "end": 5817
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5808,
                                      "end": 5817
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 5834,
                                        "end": 5843
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5834,
                                      "end": 5843
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5790,
                                  "end": 5857
                                }
                              },
                              "loc": {
                                "start": 5786,
                                "end": 5857
                              }
                            }
                          ],
                          "loc": {
                            "start": 4607,
                            "end": 5867
                          }
                        },
                        "loc": {
                          "start": 4599,
                          "end": 5867
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5876,
                            "end": 5878
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5876,
                          "end": 5878
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5887,
                            "end": 5897
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5887,
                          "end": 5897
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5906,
                            "end": 5916
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5906,
                          "end": 5916
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5925,
                            "end": 5929
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5925,
                          "end": 5929
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 5938,
                            "end": 5949
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5938,
                          "end": 5949
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 5958,
                            "end": 5970
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5958,
                          "end": 5970
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 5979,
                            "end": 5991
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
                                  "start": 6006,
                                  "end": 6008
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6006,
                                "end": 6008
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 6021,
                                  "end": 6032
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6021,
                                "end": 6032
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 6045,
                                  "end": 6051
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6045,
                                "end": 6051
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 6064,
                                  "end": 6076
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6064,
                                "end": 6076
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 6089,
                                  "end": 6092
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
                                        "start": 6111,
                                        "end": 6124
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6111,
                                      "end": 6124
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 6141,
                                        "end": 6150
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6141,
                                      "end": 6150
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 6167,
                                        "end": 6178
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6167,
                                      "end": 6178
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 6195,
                                        "end": 6204
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6195,
                                      "end": 6204
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 6221,
                                        "end": 6230
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6221,
                                      "end": 6230
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 6247,
                                        "end": 6254
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6247,
                                      "end": 6254
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 6271,
                                        "end": 6283
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6271,
                                      "end": 6283
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 6300,
                                        "end": 6308
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6300,
                                      "end": 6308
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 6325,
                                        "end": 6339
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
                                              "start": 6362,
                                              "end": 6364
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6362,
                                            "end": 6364
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6385,
                                              "end": 6395
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6385,
                                            "end": 6395
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6416,
                                              "end": 6426
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6416,
                                            "end": 6426
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 6447,
                                              "end": 6454
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6447,
                                            "end": 6454
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 6475,
                                              "end": 6486
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6475,
                                            "end": 6486
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6340,
                                        "end": 6504
                                      }
                                    },
                                    "loc": {
                                      "start": 6325,
                                      "end": 6504
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6093,
                                  "end": 6518
                                }
                              },
                              "loc": {
                                "start": 6089,
                                "end": 6518
                              }
                            }
                          ],
                          "loc": {
                            "start": 5992,
                            "end": 6528
                          }
                        },
                        "loc": {
                          "start": 5979,
                          "end": 6528
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6537,
                            "end": 6549
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
                                  "start": 6564,
                                  "end": 6566
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6564,
                                "end": 6566
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6579,
                                  "end": 6587
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6579,
                                "end": 6587
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6600,
                                  "end": 6611
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6600,
                                "end": 6611
                              }
                            }
                          ],
                          "loc": {
                            "start": 6550,
                            "end": 6621
                          }
                        },
                        "loc": {
                          "start": 6537,
                          "end": 6621
                        }
                      }
                    ],
                    "loc": {
                      "start": 4589,
                      "end": 6627
                    }
                  },
                  "loc": {
                    "start": 4571,
                    "end": 6627
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 6632,
                      "end": 6646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6632,
                    "end": 6646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 6651,
                      "end": 6663
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6651,
                    "end": 6663
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6668,
                      "end": 6671
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
                            "start": 6682,
                            "end": 6691
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6682,
                          "end": 6691
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 6700,
                            "end": 6709
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6700,
                          "end": 6709
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6718,
                            "end": 6727
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6718,
                          "end": 6727
                        }
                      }
                    ],
                    "loc": {
                      "start": 6672,
                      "end": 6733
                    }
                  },
                  "loc": {
                    "start": 6668,
                    "end": 6733
                  }
                }
              ],
              "loc": {
                "start": 3865,
                "end": 6735
              }
            },
            "loc": {
              "start": 3856,
              "end": 6735
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 6736,
                "end": 6747
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
                      "start": 6754,
                      "end": 6768
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
                            "start": 6779,
                            "end": 6781
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6779,
                          "end": 6781
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6790,
                            "end": 6800
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6790,
                          "end": 6800
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6809,
                            "end": 6817
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6809,
                          "end": 6817
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6826,
                            "end": 6835
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6826,
                          "end": 6835
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6844,
                            "end": 6856
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6844,
                          "end": 6856
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6865,
                            "end": 6877
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6865,
                          "end": 6877
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6886,
                            "end": 6890
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
                                  "start": 6905,
                                  "end": 6907
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6905,
                                "end": 6907
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6920,
                                  "end": 6929
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6920,
                                "end": 6929
                              }
                            }
                          ],
                          "loc": {
                            "start": 6891,
                            "end": 6939
                          }
                        },
                        "loc": {
                          "start": 6886,
                          "end": 6939
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6948,
                            "end": 6960
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
                                  "start": 6975,
                                  "end": 6977
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6975,
                                "end": 6977
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6990,
                                  "end": 6998
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6990,
                                "end": 6998
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7011,
                                  "end": 7022
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7011,
                                "end": 7022
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7035,
                                  "end": 7039
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7035,
                                "end": 7039
                              }
                            }
                          ],
                          "loc": {
                            "start": 6961,
                            "end": 7049
                          }
                        },
                        "loc": {
                          "start": 6948,
                          "end": 7049
                        }
                      }
                    ],
                    "loc": {
                      "start": 6769,
                      "end": 7055
                    }
                  },
                  "loc": {
                    "start": 6754,
                    "end": 7055
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7060,
                      "end": 7062
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7060,
                    "end": 7062
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7067,
                      "end": 7076
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7067,
                    "end": 7076
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 7081,
                      "end": 7100
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7081,
                    "end": 7100
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 7105,
                      "end": 7120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7105,
                    "end": 7120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7125,
                      "end": 7134
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7125,
                    "end": 7134
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7139,
                      "end": 7150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7139,
                    "end": 7150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7155,
                      "end": 7166
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7155,
                    "end": 7166
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7171,
                      "end": 7175
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7171,
                    "end": 7175
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7180,
                      "end": 7186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7180,
                    "end": 7186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7191,
                      "end": 7201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7191,
                    "end": 7201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7206,
                      "end": 7218
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
                            "start": 7232,
                            "end": 7248
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7229,
                          "end": 7248
                        }
                      }
                    ],
                    "loc": {
                      "start": 7219,
                      "end": 7254
                    }
                  },
                  "loc": {
                    "start": 7206,
                    "end": 7254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 7259,
                      "end": 7263
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
                            "start": 7277,
                            "end": 7285
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7274,
                          "end": 7285
                        }
                      }
                    ],
                    "loc": {
                      "start": 7264,
                      "end": 7291
                    }
                  },
                  "loc": {
                    "start": 7259,
                    "end": 7291
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7296,
                      "end": 7299
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
                            "start": 7310,
                            "end": 7319
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7310,
                          "end": 7319
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7328,
                            "end": 7337
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7328,
                          "end": 7337
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7346,
                            "end": 7353
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7346,
                          "end": 7353
                        }
                      }
                    ],
                    "loc": {
                      "start": 7300,
                      "end": 7359
                    }
                  },
                  "loc": {
                    "start": 7296,
                    "end": 7359
                  }
                }
              ],
              "loc": {
                "start": 6748,
                "end": 7361
              }
            },
            "loc": {
              "start": 6736,
              "end": 7361
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 7362,
                "end": 7373
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
                      "start": 7380,
                      "end": 7394
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
                            "start": 7405,
                            "end": 7407
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7405,
                          "end": 7407
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 7416,
                            "end": 7426
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7416,
                          "end": 7426
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 7435,
                            "end": 7448
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7435,
                          "end": 7448
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 7457,
                            "end": 7467
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7457,
                          "end": 7467
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 7476,
                            "end": 7485
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7476,
                          "end": 7485
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 7494,
                            "end": 7502
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7494,
                          "end": 7502
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7511,
                            "end": 7520
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7511,
                          "end": 7520
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 7529,
                            "end": 7533
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
                                  "start": 7548,
                                  "end": 7550
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7548,
                                "end": 7550
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 7563,
                                  "end": 7573
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7563,
                                "end": 7573
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 7586,
                                  "end": 7595
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7586,
                                "end": 7595
                              }
                            }
                          ],
                          "loc": {
                            "start": 7534,
                            "end": 7605
                          }
                        },
                        "loc": {
                          "start": 7529,
                          "end": 7605
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 7614,
                            "end": 7626
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
                                  "start": 7641,
                                  "end": 7643
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7641,
                                "end": 7643
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 7656,
                                  "end": 7664
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7656,
                                "end": 7664
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7677,
                                  "end": 7688
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7677,
                                "end": 7688
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 7701,
                                  "end": 7713
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7701,
                                "end": 7713
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7726,
                                  "end": 7730
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7726,
                                "end": 7730
                              }
                            }
                          ],
                          "loc": {
                            "start": 7627,
                            "end": 7740
                          }
                        },
                        "loc": {
                          "start": 7614,
                          "end": 7740
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 7749,
                            "end": 7761
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7749,
                          "end": 7761
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 7770,
                            "end": 7782
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7770,
                          "end": 7782
                        }
                      }
                    ],
                    "loc": {
                      "start": 7395,
                      "end": 7788
                    }
                  },
                  "loc": {
                    "start": 7380,
                    "end": 7788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7793,
                      "end": 7795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7793,
                    "end": 7795
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7800,
                      "end": 7809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7800,
                    "end": 7809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 7814,
                      "end": 7833
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7814,
                    "end": 7833
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 7838,
                      "end": 7853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7838,
                    "end": 7853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7858,
                      "end": 7867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7858,
                    "end": 7867
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7872,
                      "end": 7883
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7872,
                    "end": 7883
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7888,
                      "end": 7899
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7888,
                    "end": 7899
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7904,
                      "end": 7908
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7904,
                    "end": 7908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7913,
                      "end": 7919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7913,
                    "end": 7919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7924,
                      "end": 7934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7924,
                    "end": 7934
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 7939,
                      "end": 7950
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7939,
                    "end": 7950
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 7955,
                      "end": 7974
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7955,
                    "end": 7974
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7979,
                      "end": 7991
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
                            "start": 8005,
                            "end": 8021
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 8002,
                          "end": 8021
                        }
                      }
                    ],
                    "loc": {
                      "start": 7992,
                      "end": 8027
                    }
                  },
                  "loc": {
                    "start": 7979,
                    "end": 8027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 8032,
                      "end": 8036
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
                            "start": 8050,
                            "end": 8058
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 8047,
                          "end": 8058
                        }
                      }
                    ],
                    "loc": {
                      "start": 8037,
                      "end": 8064
                    }
                  },
                  "loc": {
                    "start": 8032,
                    "end": 8064
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 8069,
                      "end": 8072
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
                            "start": 8083,
                            "end": 8092
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8083,
                          "end": 8092
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 8101,
                            "end": 8110
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8101,
                          "end": 8110
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 8119,
                            "end": 8126
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8119,
                          "end": 8126
                        }
                      }
                    ],
                    "loc": {
                      "start": 8073,
                      "end": 8132
                    }
                  },
                  "loc": {
                    "start": 8069,
                    "end": 8132
                  }
                }
              ],
              "loc": {
                "start": 7374,
                "end": 8134
              }
            },
            "loc": {
              "start": 7362,
              "end": 8134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8135,
                "end": 8137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8135,
              "end": 8137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8138,
                "end": 8148
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8138,
              "end": 8148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 8149,
                "end": 8159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8149,
              "end": 8159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 8160,
                "end": 8169
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8160,
              "end": 8169
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 8170,
                "end": 8177
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8170,
              "end": 8177
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 8178,
                "end": 8186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8178,
              "end": 8186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 8187,
                "end": 8197
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
                      "start": 8204,
                      "end": 8206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8204,
                    "end": 8206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 8211,
                      "end": 8228
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8211,
                    "end": 8228
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 8233,
                      "end": 8245
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8233,
                    "end": 8245
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 8250,
                      "end": 8260
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8250,
                    "end": 8260
                  }
                }
              ],
              "loc": {
                "start": 8198,
                "end": 8262
              }
            },
            "loc": {
              "start": 8187,
              "end": 8262
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 8263,
                "end": 8274
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
                      "start": 8281,
                      "end": 8283
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8281,
                    "end": 8283
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 8288,
                      "end": 8302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8288,
                    "end": 8302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 8307,
                      "end": 8315
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8307,
                    "end": 8315
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 8320,
                      "end": 8329
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8320,
                    "end": 8329
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 8334,
                      "end": 8344
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8334,
                    "end": 8344
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 8349,
                      "end": 8354
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8349,
                    "end": 8354
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 8359,
                      "end": 8366
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8359,
                    "end": 8366
                  }
                }
              ],
              "loc": {
                "start": 8275,
                "end": 8368
              }
            },
            "loc": {
              "start": 8263,
              "end": 8368
            }
          }
        ],
        "loc": {
          "start": 2817,
          "end": 8370
        }
      },
      "loc": {
        "start": 2782,
        "end": 8370
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 8380,
          "end": 8388
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 8392,
            "end": 8395
          }
        },
        "loc": {
          "start": 8392,
          "end": 8395
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
                "start": 8398,
                "end": 8400
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8398,
              "end": 8400
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8401,
                "end": 8411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8401,
              "end": 8411
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 8412,
                "end": 8415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8412,
              "end": 8415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 8416,
                "end": 8425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8416,
              "end": 8425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 8426,
                "end": 8438
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
                      "start": 8445,
                      "end": 8447
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8445,
                    "end": 8447
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 8452,
                      "end": 8460
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8452,
                    "end": 8460
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 8465,
                      "end": 8476
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8465,
                    "end": 8476
                  }
                }
              ],
              "loc": {
                "start": 8439,
                "end": 8478
              }
            },
            "loc": {
              "start": 8426,
              "end": 8478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8479,
                "end": 8482
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
                      "start": 8489,
                      "end": 8494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8489,
                    "end": 8494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8499,
                      "end": 8511
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8499,
                    "end": 8511
                  }
                }
              ],
              "loc": {
                "start": 8483,
                "end": 8513
              }
            },
            "loc": {
              "start": 8479,
              "end": 8513
            }
          }
        ],
        "loc": {
          "start": 8396,
          "end": 8515
        }
      },
      "loc": {
        "start": 8371,
        "end": 8515
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 8525,
          "end": 8533
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8537,
            "end": 8541
          }
        },
        "loc": {
          "start": 8537,
          "end": 8541
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
                "start": 8544,
                "end": 8546
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8544,
              "end": 8546
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8547,
                "end": 8557
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8547,
              "end": 8557
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 8558,
                "end": 8568
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8558,
              "end": 8568
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 8569,
                "end": 8580
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8569,
              "end": 8580
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8581,
                "end": 8587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8581,
              "end": 8587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8588,
                "end": 8593
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8588,
              "end": 8593
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8594,
                "end": 8598
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8594,
              "end": 8598
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8599,
                "end": 8611
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8599,
              "end": 8611
            }
          }
        ],
        "loc": {
          "start": 8542,
          "end": 8613
        }
      },
      "loc": {
        "start": 8516,
        "end": 8613
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
        "start": 8621,
        "end": 8625
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
              "start": 8627,
              "end": 8632
            }
          },
          "loc": {
            "start": 8626,
            "end": 8632
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
                "start": 8634,
                "end": 8643
              }
            },
            "loc": {
              "start": 8634,
              "end": 8643
            }
          },
          "loc": {
            "start": 8634,
            "end": 8644
          }
        },
        "directives": [],
        "loc": {
          "start": 8626,
          "end": 8644
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
              "start": 8650,
              "end": 8654
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8655,
                  "end": 8660
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8663,
                    "end": 8668
                  }
                },
                "loc": {
                  "start": 8662,
                  "end": 8668
                }
              },
              "loc": {
                "start": 8655,
                "end": 8668
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
                    "start": 8676,
                    "end": 8681
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
                          "start": 8695,
                          "end": 8704
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8692,
                        "end": 8704
                      }
                    }
                  ],
                  "loc": {
                    "start": 8682,
                    "end": 8710
                  }
                },
                "loc": {
                  "start": 8676,
                  "end": 8710
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 8715,
                    "end": 8724
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
                          "start": 8738,
                          "end": 8751
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8735,
                        "end": 8751
                      }
                    }
                  ],
                  "loc": {
                    "start": 8725,
                    "end": 8757
                  }
                },
                "loc": {
                  "start": 8715,
                  "end": 8757
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 8762,
                    "end": 8771
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
                          "start": 8785,
                          "end": 8798
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8782,
                        "end": 8798
                      }
                    }
                  ],
                  "loc": {
                    "start": 8772,
                    "end": 8804
                  }
                },
                "loc": {
                  "start": 8762,
                  "end": 8804
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 8809,
                    "end": 8818
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
                          "start": 8832,
                          "end": 8845
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8829,
                        "end": 8845
                      }
                    }
                  ],
                  "loc": {
                    "start": 8819,
                    "end": 8851
                  }
                },
                "loc": {
                  "start": 8809,
                  "end": 8851
                }
              }
            ],
            "loc": {
              "start": 8670,
              "end": 8855
            }
          },
          "loc": {
            "start": 8650,
            "end": 8855
          }
        }
      ],
      "loc": {
        "start": 8646,
        "end": 8857
      }
    },
    "loc": {
      "start": 8615,
      "end": 8857
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
