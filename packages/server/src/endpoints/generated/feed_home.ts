export const feed_home = {
  "fieldName": "home",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "home",
        "loc": {
          "start": 8536,
          "end": 8540
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8541,
              "end": 8546
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8549,
                "end": 8554
              }
            },
            "loc": {
              "start": 8548,
              "end": 8554
            }
          },
          "loc": {
            "start": 8541,
            "end": 8554
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
                "start": 8562,
                "end": 8567
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
                      "start": 8581,
                      "end": 8590
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8578,
                    "end": 8590
                  }
                }
              ],
              "loc": {
                "start": 8568,
                "end": 8596
              }
            },
            "loc": {
              "start": 8562,
              "end": 8596
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reminders",
              "loc": {
                "start": 8601,
                "end": 8610
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
                      "start": 8624,
                      "end": 8637
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8621,
                    "end": 8637
                  }
                }
              ],
              "loc": {
                "start": 8611,
                "end": 8643
              }
            },
            "loc": {
              "start": 8601,
              "end": 8643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "resources",
              "loc": {
                "start": 8648,
                "end": 8657
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
                      "start": 8671,
                      "end": 8684
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8668,
                    "end": 8684
                  }
                }
              ],
              "loc": {
                "start": 8658,
                "end": 8690
              }
            },
            "loc": {
              "start": 8648,
              "end": 8690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedules",
              "loc": {
                "start": 8695,
                "end": 8704
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
                      "start": 8718,
                      "end": 8731
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8715,
                    "end": 8731
                  }
                }
              ],
              "loc": {
                "start": 8705,
                "end": 8737
              }
            },
            "loc": {
              "start": 8695,
              "end": 8737
            }
          }
        ],
        "loc": {
          "start": 8556,
          "end": 8741
        }
      },
      "loc": {
        "start": 8536,
        "end": 8741
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
                      }
                    ],
                    "loc": {
                      "start": 4577,
                      "end": 5753
                    }
                  },
                  "loc": {
                    "start": 4569,
                    "end": 5753
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5762,
                      "end": 5764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5762,
                    "end": 5764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5773,
                      "end": 5783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5773,
                    "end": 5783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5792,
                      "end": 5802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5792,
                    "end": 5802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5811,
                      "end": 5815
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5811,
                    "end": 5815
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 5824,
                      "end": 5835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5824,
                    "end": 5835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membersCount",
                    "loc": {
                      "start": 5844,
                      "end": 5856
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5844,
                    "end": 5856
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 5865,
                      "end": 5877
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
                            "start": 5892,
                            "end": 5894
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5892,
                          "end": 5894
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bannerImage",
                          "loc": {
                            "start": 5907,
                            "end": 5918
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5907,
                          "end": 5918
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "handle",
                          "loc": {
                            "start": 5931,
                            "end": 5937
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5931,
                          "end": 5937
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "profileImage",
                          "loc": {
                            "start": 5950,
                            "end": 5962
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5950,
                          "end": 5962
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 5975,
                            "end": 5978
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
                                  "start": 5997,
                                  "end": 6010
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5997,
                                "end": 6010
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canDelete",
                                "loc": {
                                  "start": 6027,
                                  "end": 6036
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6027,
                                "end": 6036
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canBookmark",
                                "loc": {
                                  "start": 6053,
                                  "end": 6064
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6053,
                                "end": 6064
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canReport",
                                "loc": {
                                  "start": 6081,
                                  "end": 6090
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6081,
                                "end": 6090
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 6107,
                                  "end": 6116
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6107,
                                "end": 6116
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 6133,
                                  "end": 6140
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6133,
                                "end": 6140
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBookmarked",
                                "loc": {
                                  "start": 6157,
                                  "end": 6169
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6157,
                                "end": 6169
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isViewed",
                                "loc": {
                                  "start": 6186,
                                  "end": 6194
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6186,
                                "end": 6194
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "yourMembership",
                                "loc": {
                                  "start": 6211,
                                  "end": 6225
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
                                        "start": 6248,
                                        "end": 6250
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6248,
                                      "end": 6250
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 6271,
                                        "end": 6281
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6271,
                                      "end": 6281
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 6302,
                                        "end": 6312
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6302,
                                      "end": 6312
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isAdmin",
                                      "loc": {
                                        "start": 6333,
                                        "end": 6340
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6333,
                                      "end": 6340
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "permissions",
                                      "loc": {
                                        "start": 6361,
                                        "end": 6372
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6361,
                                      "end": 6372
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6226,
                                  "end": 6390
                                }
                              },
                              "loc": {
                                "start": 6211,
                                "end": 6390
                              }
                            }
                          ],
                          "loc": {
                            "start": 5979,
                            "end": 6404
                          }
                        },
                        "loc": {
                          "start": 5975,
                          "end": 6404
                        }
                      }
                    ],
                    "loc": {
                      "start": 5878,
                      "end": 6414
                    }
                  },
                  "loc": {
                    "start": 5865,
                    "end": 6414
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6423,
                      "end": 6435
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
                            "start": 6450,
                            "end": 6452
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6450,
                          "end": 6452
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6465,
                            "end": 6473
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6465,
                          "end": 6473
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6486,
                            "end": 6497
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6486,
                          "end": 6497
                        }
                      }
                    ],
                    "loc": {
                      "start": 6436,
                      "end": 6507
                    }
                  },
                  "loc": {
                    "start": 6423,
                    "end": 6507
                  }
                }
              ],
              "loc": {
                "start": 4559,
                "end": 6513
              }
            },
            "loc": {
              "start": 4541,
              "end": 6513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "attendeesCount",
              "loc": {
                "start": 6518,
                "end": 6532
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6518,
              "end": 6532
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "invitesCount",
              "loc": {
                "start": 6537,
                "end": 6549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6537,
              "end": 6549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6554,
                "end": 6557
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
                      "start": 6568,
                      "end": 6577
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6568,
                    "end": 6577
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canInvite",
                    "loc": {
                      "start": 6586,
                      "end": 6595
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6586,
                    "end": 6595
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6604,
                      "end": 6613
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6604,
                    "end": 6613
                  }
                }
              ],
              "loc": {
                "start": 6558,
                "end": 6619
              }
            },
            "loc": {
              "start": 6554,
              "end": 6619
            }
          }
        ],
        "loc": {
          "start": 3865,
          "end": 6621
        }
      },
      "loc": {
        "start": 3856,
        "end": 6621
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runProjects",
        "loc": {
          "start": 6622,
          "end": 6633
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
                "start": 6640,
                "end": 6654
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
                      "start": 6665,
                      "end": 6667
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6665,
                    "end": 6667
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 6676,
                      "end": 6686
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6676,
                    "end": 6686
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6695,
                      "end": 6703
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6695,
                    "end": 6703
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6712,
                      "end": 6721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6712,
                    "end": 6721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6730,
                      "end": 6742
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6730,
                    "end": 6742
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6751,
                      "end": 6763
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6751,
                    "end": 6763
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 6772,
                      "end": 6776
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
                            "start": 6791,
                            "end": 6793
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6791,
                          "end": 6793
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6806,
                            "end": 6815
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6806,
                          "end": 6815
                        }
                      }
                    ],
                    "loc": {
                      "start": 6777,
                      "end": 6825
                    }
                  },
                  "loc": {
                    "start": 6772,
                    "end": 6825
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 6834,
                      "end": 6846
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
                            "start": 6861,
                            "end": 6863
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6861,
                          "end": 6863
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6876,
                            "end": 6884
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6876,
                          "end": 6884
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6897,
                            "end": 6908
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6897,
                          "end": 6908
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6921,
                            "end": 6925
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6921,
                          "end": 6925
                        }
                      }
                    ],
                    "loc": {
                      "start": 6847,
                      "end": 6935
                    }
                  },
                  "loc": {
                    "start": 6834,
                    "end": 6935
                  }
                }
              ],
              "loc": {
                "start": 6655,
                "end": 6941
              }
            },
            "loc": {
              "start": 6640,
              "end": 6941
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6946,
                "end": 6948
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6946,
              "end": 6948
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6953,
                "end": 6962
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6953,
              "end": 6962
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 6967,
                "end": 6986
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6967,
              "end": 6986
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 6991,
                "end": 7006
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6991,
              "end": 7006
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7011,
                "end": 7020
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7011,
              "end": 7020
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7025,
                "end": 7036
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7025,
              "end": 7036
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7041,
                "end": 7052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7041,
              "end": 7052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7057,
                "end": 7061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7057,
              "end": 7061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7066,
                "end": 7072
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7066,
              "end": 7072
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7077,
                "end": 7087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7077,
              "end": 7087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7092,
                "end": 7104
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
                      "start": 7118,
                      "end": 7134
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7115,
                    "end": 7134
                  }
                }
              ],
              "loc": {
                "start": 7105,
                "end": 7140
              }
            },
            "loc": {
              "start": 7092,
              "end": 7140
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 7145,
                "end": 7149
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
                      "start": 7163,
                      "end": 7171
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7160,
                    "end": 7171
                  }
                }
              ],
              "loc": {
                "start": 7150,
                "end": 7177
              }
            },
            "loc": {
              "start": 7145,
              "end": 7177
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7182,
                "end": 7185
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
                      "start": 7196,
                      "end": 7205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7196,
                    "end": 7205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7214,
                      "end": 7223
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7214,
                    "end": 7223
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7232,
                      "end": 7239
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7232,
                    "end": 7239
                  }
                }
              ],
              "loc": {
                "start": 7186,
                "end": 7245
              }
            },
            "loc": {
              "start": 7182,
              "end": 7245
            }
          }
        ],
        "loc": {
          "start": 6634,
          "end": 7247
        }
      },
      "loc": {
        "start": 6622,
        "end": 7247
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "runRoutines",
        "loc": {
          "start": 7248,
          "end": 7259
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
                "start": 7266,
                "end": 7280
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
                      "start": 7291,
                      "end": 7293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7291,
                    "end": 7293
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "complexity",
                    "loc": {
                      "start": 7302,
                      "end": 7312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7302,
                    "end": 7312
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 7321,
                      "end": 7334
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7321,
                    "end": 7334
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 7343,
                      "end": 7353
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7343,
                    "end": 7353
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 7362,
                      "end": 7371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7362,
                    "end": 7371
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 7380,
                      "end": 7388
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7380,
                    "end": 7388
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7397,
                      "end": 7406
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7397,
                    "end": 7406
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "root",
                    "loc": {
                      "start": 7415,
                      "end": 7419
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
                            "start": 7434,
                            "end": 7436
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7434,
                          "end": 7436
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isInternal",
                          "loc": {
                            "start": 7449,
                            "end": 7459
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7449,
                          "end": 7459
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7472,
                            "end": 7481
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7472,
                          "end": 7481
                        }
                      }
                    ],
                    "loc": {
                      "start": 7420,
                      "end": 7491
                    }
                  },
                  "loc": {
                    "start": 7415,
                    "end": 7491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "translations",
                    "loc": {
                      "start": 7500,
                      "end": 7512
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
                            "start": 7527,
                            "end": 7529
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7527,
                          "end": 7529
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 7542,
                            "end": 7550
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7542,
                          "end": 7550
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7563,
                            "end": 7574
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7563,
                          "end": 7574
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 7587,
                            "end": 7599
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7587,
                          "end": 7599
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7612,
                            "end": 7616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7612,
                          "end": 7616
                        }
                      }
                    ],
                    "loc": {
                      "start": 7513,
                      "end": 7626
                    }
                  },
                  "loc": {
                    "start": 7500,
                    "end": 7626
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 7635,
                      "end": 7647
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7635,
                    "end": 7647
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 7656,
                      "end": 7668
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7656,
                    "end": 7668
                  }
                }
              ],
              "loc": {
                "start": 7281,
                "end": 7674
              }
            },
            "loc": {
              "start": 7266,
              "end": 7674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7679,
                "end": 7681
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7679,
              "end": 7681
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7686,
                "end": 7695
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7686,
              "end": 7695
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedComplexity",
              "loc": {
                "start": 7700,
                "end": 7719
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7700,
              "end": 7719
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contextSwitches",
              "loc": {
                "start": 7724,
                "end": 7739
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7724,
              "end": 7739
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startedAt",
              "loc": {
                "start": 7744,
                "end": 7753
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7744,
              "end": 7753
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeElapsed",
              "loc": {
                "start": 7758,
                "end": 7769
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7758,
              "end": 7769
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 7774,
                "end": 7785
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7774,
              "end": 7785
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7790,
                "end": 7794
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7790,
              "end": 7794
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "status",
              "loc": {
                "start": 7799,
                "end": 7805
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7799,
              "end": 7805
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stepsCount",
              "loc": {
                "start": 7810,
                "end": 7820
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7810,
              "end": 7820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 7825,
                "end": 7836
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7825,
              "end": 7836
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wasRunAutomatically",
              "loc": {
                "start": 7841,
                "end": 7860
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7841,
              "end": 7860
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organization",
              "loc": {
                "start": 7865,
                "end": 7877
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
                      "start": 7891,
                      "end": 7907
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7888,
                    "end": 7907
                  }
                }
              ],
              "loc": {
                "start": 7878,
                "end": 7913
              }
            },
            "loc": {
              "start": 7865,
              "end": 7913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "user",
              "loc": {
                "start": 7918,
                "end": 7922
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
                      "start": 7936,
                      "end": 7944
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7933,
                    "end": 7944
                  }
                }
              ],
              "loc": {
                "start": 7923,
                "end": 7950
              }
            },
            "loc": {
              "start": 7918,
              "end": 7950
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7955,
                "end": 7958
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
                      "start": 7969,
                      "end": 7978
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7969,
                    "end": 7978
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7987,
                      "end": 7996
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7987,
                    "end": 7996
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 8005,
                      "end": 8012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8005,
                    "end": 8012
                  }
                }
              ],
              "loc": {
                "start": 7959,
                "end": 8018
              }
            },
            "loc": {
              "start": 7955,
              "end": 8018
            }
          }
        ],
        "loc": {
          "start": 7260,
          "end": 8020
        }
      },
      "loc": {
        "start": 7248,
        "end": 8020
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8021,
          "end": 8023
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8021,
        "end": 8023
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8024,
          "end": 8034
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8024,
        "end": 8034
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 8035,
          "end": 8045
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8035,
        "end": 8045
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 8046,
          "end": 8055
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8046,
        "end": 8055
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 8056,
          "end": 8063
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8056,
        "end": 8063
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 8064,
          "end": 8072
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8064,
        "end": 8072
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 8073,
          "end": 8083
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
                "start": 8090,
                "end": 8092
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8090,
              "end": 8092
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 8097,
                "end": 8114
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8097,
              "end": 8114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 8119,
                "end": 8131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8119,
              "end": 8131
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 8136,
                "end": 8146
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8136,
              "end": 8146
            }
          }
        ],
        "loc": {
          "start": 8084,
          "end": 8148
        }
      },
      "loc": {
        "start": 8073,
        "end": 8148
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 8149,
          "end": 8160
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
                "start": 8167,
                "end": 8169
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8167,
              "end": 8169
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 8174,
                "end": 8188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8174,
              "end": 8188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 8193,
                "end": 8201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8193,
              "end": 8201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 8206,
                "end": 8215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8206,
              "end": 8215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
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
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 8235,
                "end": 8240
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8235,
              "end": 8240
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 8245,
                "end": 8252
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8245,
              "end": 8252
            }
          }
        ],
        "loc": {
          "start": 8161,
          "end": 8254
        }
      },
      "loc": {
        "start": 8149,
        "end": 8254
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8284,
          "end": 8286
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8284,
        "end": 8286
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8287,
          "end": 8297
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8287,
        "end": 8297
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 8298,
          "end": 8301
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8298,
        "end": 8301
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 8302,
          "end": 8311
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8302,
        "end": 8311
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 8312,
          "end": 8324
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
                "start": 8331,
                "end": 8333
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8331,
              "end": 8333
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 8338,
                "end": 8346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8338,
              "end": 8346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 8351,
                "end": 8362
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8351,
              "end": 8362
            }
          }
        ],
        "loc": {
          "start": 8325,
          "end": 8364
        }
      },
      "loc": {
        "start": 8312,
        "end": 8364
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8365,
          "end": 8368
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
                "start": 8375,
                "end": 8380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8375,
              "end": 8380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8385,
                "end": 8397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8385,
              "end": 8397
            }
          }
        ],
        "loc": {
          "start": 8369,
          "end": 8399
        }
      },
      "loc": {
        "start": 8365,
        "end": 8399
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8430,
          "end": 8432
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8430,
        "end": 8432
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8433,
          "end": 8443
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8433,
        "end": 8443
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 8444,
          "end": 8454
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8444,
        "end": 8454
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 8455,
          "end": 8466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8455,
        "end": 8466
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8467,
          "end": 8473
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8467,
        "end": 8473
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8474,
          "end": 8479
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8474,
        "end": 8479
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8480,
          "end": 8484
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8480,
        "end": 8484
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8485,
          "end": 8497
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8485,
        "end": 8497
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
                            }
                          ],
                          "loc": {
                            "start": 4577,
                            "end": 5753
                          }
                        },
                        "loc": {
                          "start": 4569,
                          "end": 5753
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 5762,
                            "end": 5764
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5762,
                          "end": 5764
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5773,
                            "end": 5783
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5773,
                          "end": 5783
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5792,
                            "end": 5802
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5792,
                          "end": 5802
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5811,
                            "end": 5815
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5811,
                          "end": 5815
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 5824,
                            "end": 5835
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5824,
                          "end": 5835
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "membersCount",
                          "loc": {
                            "start": 5844,
                            "end": 5856
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5844,
                          "end": 5856
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "organization",
                          "loc": {
                            "start": 5865,
                            "end": 5877
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
                                  "start": 5892,
                                  "end": 5894
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5892,
                                "end": 5894
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 5907,
                                  "end": 5918
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5907,
                                "end": 5918
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 5931,
                                  "end": 5937
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5931,
                                "end": 5937
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 5950,
                                  "end": 5962
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5950,
                                "end": 5962
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5975,
                                  "end": 5978
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
                                        "start": 5997,
                                        "end": 6010
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5997,
                                      "end": 6010
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 6027,
                                        "end": 6036
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6027,
                                      "end": 6036
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canBookmark",
                                      "loc": {
                                        "start": 6053,
                                        "end": 6064
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6053,
                                      "end": 6064
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canReport",
                                      "loc": {
                                        "start": 6081,
                                        "end": 6090
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6081,
                                      "end": 6090
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 6107,
                                        "end": 6116
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6107,
                                      "end": 6116
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 6133,
                                        "end": 6140
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6133,
                                      "end": 6140
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 6157,
                                        "end": 6169
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6157,
                                      "end": 6169
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isViewed",
                                      "loc": {
                                        "start": 6186,
                                        "end": 6194
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6186,
                                      "end": 6194
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "yourMembership",
                                      "loc": {
                                        "start": 6211,
                                        "end": 6225
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
                                              "start": 6248,
                                              "end": 6250
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6248,
                                            "end": 6250
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6271,
                                              "end": 6281
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6271,
                                            "end": 6281
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6302,
                                              "end": 6312
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6302,
                                            "end": 6312
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isAdmin",
                                            "loc": {
                                              "start": 6333,
                                              "end": 6340
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6333,
                                            "end": 6340
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "permissions",
                                            "loc": {
                                              "start": 6361,
                                              "end": 6372
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6361,
                                            "end": 6372
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6226,
                                        "end": 6390
                                      }
                                    },
                                    "loc": {
                                      "start": 6211,
                                      "end": 6390
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5979,
                                  "end": 6404
                                }
                              },
                              "loc": {
                                "start": 5975,
                                "end": 6404
                              }
                            }
                          ],
                          "loc": {
                            "start": 5878,
                            "end": 6414
                          }
                        },
                        "loc": {
                          "start": 5865,
                          "end": 6414
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6423,
                            "end": 6435
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
                                  "start": 6450,
                                  "end": 6452
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6450,
                                "end": 6452
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6465,
                                  "end": 6473
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6465,
                                "end": 6473
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6486,
                                  "end": 6497
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6486,
                                "end": 6497
                              }
                            }
                          ],
                          "loc": {
                            "start": 6436,
                            "end": 6507
                          }
                        },
                        "loc": {
                          "start": 6423,
                          "end": 6507
                        }
                      }
                    ],
                    "loc": {
                      "start": 4559,
                      "end": 6513
                    }
                  },
                  "loc": {
                    "start": 4541,
                    "end": 6513
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "attendeesCount",
                    "loc": {
                      "start": 6518,
                      "end": 6532
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6518,
                    "end": 6532
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "invitesCount",
                    "loc": {
                      "start": 6537,
                      "end": 6549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6537,
                    "end": 6549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6554,
                      "end": 6557
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
                            "start": 6568,
                            "end": 6577
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6568,
                          "end": 6577
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canInvite",
                          "loc": {
                            "start": 6586,
                            "end": 6595
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6586,
                          "end": 6595
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6604,
                            "end": 6613
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6604,
                          "end": 6613
                        }
                      }
                    ],
                    "loc": {
                      "start": 6558,
                      "end": 6619
                    }
                  },
                  "loc": {
                    "start": 6554,
                    "end": 6619
                  }
                }
              ],
              "loc": {
                "start": 3865,
                "end": 6621
              }
            },
            "loc": {
              "start": 3856,
              "end": 6621
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjects",
              "loc": {
                "start": 6622,
                "end": 6633
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
                      "start": 6640,
                      "end": 6654
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
                            "start": 6665,
                            "end": 6667
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6665,
                          "end": 6667
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 6676,
                            "end": 6686
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6676,
                          "end": 6686
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 6695,
                            "end": 6703
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6695,
                          "end": 6703
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 6712,
                            "end": 6721
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6712,
                          "end": 6721
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 6730,
                            "end": 6742
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6730,
                          "end": 6742
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 6751,
                            "end": 6763
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6751,
                          "end": 6763
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 6772,
                            "end": 6776
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
                                  "start": 6791,
                                  "end": 6793
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6791,
                                "end": 6793
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 6806,
                                  "end": 6815
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6806,
                                "end": 6815
                              }
                            }
                          ],
                          "loc": {
                            "start": 6777,
                            "end": 6825
                          }
                        },
                        "loc": {
                          "start": 6772,
                          "end": 6825
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 6834,
                            "end": 6846
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
                                  "start": 6861,
                                  "end": 6863
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6861,
                                "end": 6863
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 6876,
                                  "end": 6884
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6876,
                                "end": 6884
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 6897,
                                  "end": 6908
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6897,
                                "end": 6908
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 6921,
                                  "end": 6925
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6921,
                                "end": 6925
                              }
                            }
                          ],
                          "loc": {
                            "start": 6847,
                            "end": 6935
                          }
                        },
                        "loc": {
                          "start": 6834,
                          "end": 6935
                        }
                      }
                    ],
                    "loc": {
                      "start": 6655,
                      "end": 6941
                    }
                  },
                  "loc": {
                    "start": 6640,
                    "end": 6941
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6946,
                      "end": 6948
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6946,
                    "end": 6948
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6953,
                      "end": 6962
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6953,
                    "end": 6962
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 6967,
                      "end": 6986
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6967,
                    "end": 6986
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 6991,
                      "end": 7006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6991,
                    "end": 7006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7011,
                      "end": 7020
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7011,
                    "end": 7020
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7025,
                      "end": 7036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7025,
                    "end": 7036
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7041,
                      "end": 7052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7041,
                    "end": 7052
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7057,
                      "end": 7061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7057,
                    "end": 7061
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7066,
                      "end": 7072
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7066,
                    "end": 7072
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7077,
                      "end": 7087
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7077,
                    "end": 7087
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7092,
                      "end": 7104
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
                            "start": 7118,
                            "end": 7134
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7115,
                          "end": 7134
                        }
                      }
                    ],
                    "loc": {
                      "start": 7105,
                      "end": 7140
                    }
                  },
                  "loc": {
                    "start": 7092,
                    "end": 7140
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 7145,
                      "end": 7149
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
                            "start": 7163,
                            "end": 7171
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7160,
                          "end": 7171
                        }
                      }
                    ],
                    "loc": {
                      "start": 7150,
                      "end": 7177
                    }
                  },
                  "loc": {
                    "start": 7145,
                    "end": 7177
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7182,
                      "end": 7185
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
                            "start": 7196,
                            "end": 7205
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7196,
                          "end": 7205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7214,
                            "end": 7223
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7214,
                          "end": 7223
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7232,
                            "end": 7239
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7232,
                          "end": 7239
                        }
                      }
                    ],
                    "loc": {
                      "start": 7186,
                      "end": 7245
                    }
                  },
                  "loc": {
                    "start": 7182,
                    "end": 7245
                  }
                }
              ],
              "loc": {
                "start": 6634,
                "end": 7247
              }
            },
            "loc": {
              "start": 6622,
              "end": 7247
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runRoutines",
              "loc": {
                "start": 7248,
                "end": 7259
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
                      "start": 7266,
                      "end": 7280
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
                            "start": 7291,
                            "end": 7293
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7291,
                          "end": 7293
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "complexity",
                          "loc": {
                            "start": 7302,
                            "end": 7312
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7302,
                          "end": 7312
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAutomatable",
                          "loc": {
                            "start": 7321,
                            "end": 7334
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7321,
                          "end": 7334
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isComplete",
                          "loc": {
                            "start": 7343,
                            "end": 7353
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7343,
                          "end": 7353
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isDeleted",
                          "loc": {
                            "start": 7362,
                            "end": 7371
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7362,
                          "end": 7371
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isLatest",
                          "loc": {
                            "start": 7380,
                            "end": 7388
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7380,
                          "end": 7388
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isPrivate",
                          "loc": {
                            "start": 7397,
                            "end": 7406
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7397,
                          "end": 7406
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "root",
                          "loc": {
                            "start": 7415,
                            "end": 7419
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
                                  "start": 7434,
                                  "end": 7436
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7434,
                                "end": 7436
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isInternal",
                                "loc": {
                                  "start": 7449,
                                  "end": 7459
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7449,
                                "end": 7459
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isPrivate",
                                "loc": {
                                  "start": 7472,
                                  "end": 7481
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7472,
                                "end": 7481
                              }
                            }
                          ],
                          "loc": {
                            "start": 7420,
                            "end": 7491
                          }
                        },
                        "loc": {
                          "start": 7415,
                          "end": 7491
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 7500,
                            "end": 7512
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
                                  "start": 7527,
                                  "end": 7529
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7527,
                                "end": 7529
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 7542,
                                  "end": 7550
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7542,
                                "end": 7550
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7563,
                                  "end": 7574
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7563,
                                "end": 7574
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "instructions",
                                "loc": {
                                  "start": 7587,
                                  "end": 7599
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7587,
                                "end": 7599
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7612,
                                  "end": 7616
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7612,
                                "end": 7616
                              }
                            }
                          ],
                          "loc": {
                            "start": 7513,
                            "end": 7626
                          }
                        },
                        "loc": {
                          "start": 7500,
                          "end": 7626
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionIndex",
                          "loc": {
                            "start": 7635,
                            "end": 7647
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7635,
                          "end": 7647
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "versionLabel",
                          "loc": {
                            "start": 7656,
                            "end": 7668
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7656,
                          "end": 7668
                        }
                      }
                    ],
                    "loc": {
                      "start": 7281,
                      "end": 7674
                    }
                  },
                  "loc": {
                    "start": 7266,
                    "end": 7674
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7679,
                      "end": 7681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7679,
                    "end": 7681
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7686,
                      "end": 7695
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7686,
                    "end": 7695
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedComplexity",
                    "loc": {
                      "start": 7700,
                      "end": 7719
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7700,
                    "end": 7719
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contextSwitches",
                    "loc": {
                      "start": 7724,
                      "end": 7739
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7724,
                    "end": 7739
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "startedAt",
                    "loc": {
                      "start": 7744,
                      "end": 7753
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7744,
                    "end": 7753
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timeElapsed",
                    "loc": {
                      "start": 7758,
                      "end": 7769
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7758,
                    "end": 7769
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 7774,
                      "end": 7785
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7774,
                    "end": 7785
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7790,
                      "end": 7794
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7790,
                    "end": 7794
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 7799,
                      "end": 7805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7799,
                    "end": 7805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stepsCount",
                    "loc": {
                      "start": 7810,
                      "end": 7820
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7810,
                    "end": 7820
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 7825,
                      "end": 7836
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7825,
                    "end": 7836
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "wasRunAutomatically",
                    "loc": {
                      "start": 7841,
                      "end": 7860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7841,
                    "end": 7860
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "organization",
                    "loc": {
                      "start": 7865,
                      "end": 7877
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
                            "start": 7891,
                            "end": 7907
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7888,
                          "end": 7907
                        }
                      }
                    ],
                    "loc": {
                      "start": 7878,
                      "end": 7913
                    }
                  },
                  "loc": {
                    "start": 7865,
                    "end": 7913
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "user",
                    "loc": {
                      "start": 7918,
                      "end": 7922
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
                            "start": 7936,
                            "end": 7944
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7933,
                          "end": 7944
                        }
                      }
                    ],
                    "loc": {
                      "start": 7923,
                      "end": 7950
                    }
                  },
                  "loc": {
                    "start": 7918,
                    "end": 7950
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7955,
                      "end": 7958
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
                            "start": 7969,
                            "end": 7978
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7969,
                          "end": 7978
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7987,
                            "end": 7996
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7987,
                          "end": 7996
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 8005,
                            "end": 8012
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8005,
                          "end": 8012
                        }
                      }
                    ],
                    "loc": {
                      "start": 7959,
                      "end": 8018
                    }
                  },
                  "loc": {
                    "start": 7955,
                    "end": 8018
                  }
                }
              ],
              "loc": {
                "start": 7260,
                "end": 8020
              }
            },
            "loc": {
              "start": 7248,
              "end": 8020
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8021,
                "end": 8023
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8021,
              "end": 8023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8024,
                "end": 8034
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8024,
              "end": 8034
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 8035,
                "end": 8045
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8035,
              "end": 8045
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 8046,
                "end": 8055
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8046,
              "end": 8055
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 8056,
                "end": 8063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8056,
              "end": 8063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 8064,
                "end": 8072
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8064,
              "end": 8072
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 8073,
                "end": 8083
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
                      "start": 8090,
                      "end": 8092
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8090,
                    "end": 8092
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 8097,
                      "end": 8114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8097,
                    "end": 8114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 8119,
                      "end": 8131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8119,
                    "end": 8131
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 8136,
                      "end": 8146
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8136,
                    "end": 8146
                  }
                }
              ],
              "loc": {
                "start": 8084,
                "end": 8148
              }
            },
            "loc": {
              "start": 8073,
              "end": 8148
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 8149,
                "end": 8160
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
                      "start": 8167,
                      "end": 8169
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8167,
                    "end": 8169
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 8174,
                      "end": 8188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8174,
                    "end": 8188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 8193,
                      "end": 8201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8193,
                    "end": 8201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 8206,
                      "end": 8215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8206,
                    "end": 8215
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
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
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 8235,
                      "end": 8240
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8235,
                    "end": 8240
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 8245,
                      "end": 8252
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8245,
                    "end": 8252
                  }
                }
              ],
              "loc": {
                "start": 8161,
                "end": 8254
              }
            },
            "loc": {
              "start": 8149,
              "end": 8254
            }
          }
        ],
        "loc": {
          "start": 2817,
          "end": 8256
        }
      },
      "loc": {
        "start": 2782,
        "end": 8256
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 8266,
          "end": 8274
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 8278,
            "end": 8281
          }
        },
        "loc": {
          "start": 8278,
          "end": 8281
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
                "start": 8284,
                "end": 8286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8284,
              "end": 8286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8287,
                "end": 8297
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8287,
              "end": 8297
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 8298,
                "end": 8301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8298,
              "end": 8301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 8302,
                "end": 8311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8302,
              "end": 8311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 8312,
                "end": 8324
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
                      "start": 8331,
                      "end": 8333
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8331,
                    "end": 8333
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 8338,
                      "end": 8346
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8338,
                    "end": 8346
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 8351,
                      "end": 8362
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8351,
                    "end": 8362
                  }
                }
              ],
              "loc": {
                "start": 8325,
                "end": 8364
              }
            },
            "loc": {
              "start": 8312,
              "end": 8364
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8365,
                "end": 8368
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
                      "start": 8375,
                      "end": 8380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8375,
                    "end": 8380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8385,
                      "end": 8397
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8385,
                    "end": 8397
                  }
                }
              ],
              "loc": {
                "start": 8369,
                "end": 8399
              }
            },
            "loc": {
              "start": 8365,
              "end": 8399
            }
          }
        ],
        "loc": {
          "start": 8282,
          "end": 8401
        }
      },
      "loc": {
        "start": 8257,
        "end": 8401
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 8411,
          "end": 8419
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8423,
            "end": 8427
          }
        },
        "loc": {
          "start": 8423,
          "end": 8427
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
                "start": 8430,
                "end": 8432
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8430,
              "end": 8432
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8433,
                "end": 8443
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8433,
              "end": 8443
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 8444,
                "end": 8454
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8444,
              "end": 8454
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 8455,
                "end": 8466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8455,
              "end": 8466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8467,
                "end": 8473
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8467,
              "end": 8473
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8474,
                "end": 8479
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8474,
              "end": 8479
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8480,
                "end": 8484
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8480,
              "end": 8484
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8485,
                "end": 8497
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8485,
              "end": 8497
            }
          }
        ],
        "loc": {
          "start": 8428,
          "end": 8499
        }
      },
      "loc": {
        "start": 8402,
        "end": 8499
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
        "start": 8507,
        "end": 8511
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
              "start": 8513,
              "end": 8518
            }
          },
          "loc": {
            "start": 8512,
            "end": 8518
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
                "start": 8520,
                "end": 8529
              }
            },
            "loc": {
              "start": 8520,
              "end": 8529
            }
          },
          "loc": {
            "start": 8520,
            "end": 8530
          }
        },
        "directives": [],
        "loc": {
          "start": 8512,
          "end": 8530
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
              "start": 8536,
              "end": 8540
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8541,
                  "end": 8546
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8549,
                    "end": 8554
                  }
                },
                "loc": {
                  "start": 8548,
                  "end": 8554
                }
              },
              "loc": {
                "start": 8541,
                "end": 8554
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
                    "start": 8562,
                    "end": 8567
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
                          "start": 8581,
                          "end": 8590
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8578,
                        "end": 8590
                      }
                    }
                  ],
                  "loc": {
                    "start": 8568,
                    "end": 8596
                  }
                },
                "loc": {
                  "start": 8562,
                  "end": 8596
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "reminders",
                  "loc": {
                    "start": 8601,
                    "end": 8610
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
                          "start": 8624,
                          "end": 8637
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8621,
                        "end": 8637
                      }
                    }
                  ],
                  "loc": {
                    "start": 8611,
                    "end": 8643
                  }
                },
                "loc": {
                  "start": 8601,
                  "end": 8643
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "resources",
                  "loc": {
                    "start": 8648,
                    "end": 8657
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
                          "start": 8671,
                          "end": 8684
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8668,
                        "end": 8684
                      }
                    }
                  ],
                  "loc": {
                    "start": 8658,
                    "end": 8690
                  }
                },
                "loc": {
                  "start": 8648,
                  "end": 8690
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "schedules",
                  "loc": {
                    "start": 8695,
                    "end": 8704
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
                          "start": 8718,
                          "end": 8731
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8715,
                        "end": 8731
                      }
                    }
                  ],
                  "loc": {
                    "start": 8705,
                    "end": 8737
                  }
                },
                "loc": {
                  "start": 8695,
                  "end": 8737
                }
              }
            ],
            "loc": {
              "start": 8556,
              "end": 8741
            }
          },
          "loc": {
            "start": 8536,
            "end": 8741
          }
        }
      ],
      "loc": {
        "start": 8532,
        "end": 8743
      }
    },
    "loc": {
      "start": 8501,
      "end": 8743
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_home"
  }
} as const;
